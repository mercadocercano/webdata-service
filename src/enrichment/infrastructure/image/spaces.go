package image

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type SpacesStore struct {
	key      string
	secret   string
	endpoint string
	bucket   string
	cdnBase  string
	client   *http.Client
}

func NewSpacesStore() *SpacesStore {
	bucket := os.Getenv("SPACES_BUCKET")
	if bucket == "" {
		bucket = "mc-product-images"
	}
	return &SpacesStore{
		key:      os.Getenv("SPACES_KEY"),
		secret:   os.Getenv("SPACES_SECRET"),
		endpoint: os.Getenv("SPACES_ENDPOINT"),
		bucket:   bucket,
		cdnBase:  os.Getenv("SPACES_CDN_BASE"),
		client:   &http.Client{Timeout: 30 * time.Second},
	}
}

func (s *SpacesStore) Upload(ctx context.Context, gtin string, data []byte, mimeType string) (string, string, error) {
	key := buildObjectKey(gtin)

	if s.key == "" {
		return s.saveLocal(gtin, data, key)
	}

	return s.uploadToSpaces(ctx, key, data, mimeType)
}

func (s *SpacesStore) Exists(ctx context.Context, gtin string) (bool, string, error) {
	key := buildObjectKey(gtin)

	if s.key == "" {
		localPath := localFilePath(gtin)
		if _, err := os.Stat(localPath); err == nil {
			return true, "file://" + localPath, nil
		}
		return false, "", nil
	}

	objectURL := s.buildObjectURL(key)
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, objectURL, nil)
	if err != nil {
		return false, "", fmt.Errorf("spaces head request: %w", err)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return false, "", nil
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return true, s.publicURL(key), nil
	}
	return false, "", nil
}

func (s *SpacesStore) uploadToSpaces(ctx context.Context, key string, data []byte, mimeType string) (string, string, error) {
	objectURL := s.buildObjectURL(key)

	now := time.Now().UTC()
	dateStr := now.Format("20060102")
	amzDate := now.Format("20060102T150405Z")

	region := s.extractRegion()
	payloadHash := hashSHA256(data)
	headers := map[string]string{
		"content-type":         mimeType,
		"host":                 s.extractHost(),
		"x-amz-acl":            "public-read",
		"x-amz-content-sha256": payloadHash,
		"x-amz-date":           amzDate,
	}

	signedHeaders := "content-type;host;x-amz-acl;x-amz-content-sha256;x-amz-date"
	canonicalHeaders := buildCanonicalHeaders(headers, []string{"content-type", "host", "x-amz-acl", "x-amz-content-sha256", "x-amz-date"})
	canonicalRequest := buildCanonicalRequest("PUT", "/"+s.bucket+"/"+key, "", canonicalHeaders, signedHeaders, payloadHash)

	credentialScope := fmt.Sprintf("%s/%s/s3/aws4_request", dateStr, region)
	stringToSign := buildStringToSign(amzDate, credentialScope, canonicalRequest)
	signature := sign(s.secret, dateStr, region, "s3", stringToSign)

	authHeader := fmt.Sprintf(
		"AWS4-HMAC-SHA256 Credential=%s/%s,SignedHeaders=%s,Signature=%s",
		s.key, credentialScope, signedHeaders, signature,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, objectURL, bytes.NewReader(data))
	if err != nil {
		return "", "", fmt.Errorf("spaces creating upload request: %w", err)
	}

	req.Header.Set("Content-Type", mimeType)
	req.Header.Set("x-amz-date", amzDate)
	req.Header.Set("x-amz-acl", "public-read")
	req.Header.Set("x-amz-content-sha256", payloadHash)
	req.Header.Set("Authorization", authHeader)
	req.ContentLength = int64(len(data))

	resp, err := s.client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("spaces upload failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return "", "", fmt.Errorf("spaces upload status %d: %s", resp.StatusCode, string(body))
	}

	return s.publicURL(key), key, nil
}

func (s *SpacesStore) saveLocal(gtin string, data []byte, key string) (string, string, error) {
	dir := filepath.Dir(localFilePath(gtin))
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", "", fmt.Errorf("creating local dir: %w", err)
	}
	localPath := localFilePath(gtin)
	if err := os.WriteFile(localPath, data, 0644); err != nil {
		return "", "", fmt.Errorf("writing local file: %w", err)
	}
	return "file://" + localPath, key, nil
}

func (s *SpacesStore) buildObjectURL(key string) string {
	return fmt.Sprintf("%s/%s/%s", strings.TrimRight(s.endpoint, "/"), s.bucket, key)
}

func (s *SpacesStore) publicURL(key string) string {
	if s.cdnBase != "" {
		return fmt.Sprintf("%s/%s", strings.TrimRight(s.cdnBase, "/"), key)
	}
	return s.buildObjectURL(key)
}

func (s *SpacesStore) extractRegion() string {
	// virtual-hosted: mc-product-images.nyc3.digitaloceanspaces.com → nyc3
	// path-style:     nyc3.digitaloceanspaces.com → nyc3
	host := s.extractHost()
	parts := strings.Split(host, ".")
	for _, p := range parts {
		if len(p) == 4 && p[3] >= '0' && p[3] <= '9' {
			return p // e.g. nyc3, sfo3, ams3
		}
	}
	if len(parts) > 0 {
		return parts[0]
	}
	return "nyc3"
}

func (s *SpacesStore) extractHost() string {
	ep := strings.TrimPrefix(s.endpoint, "https://")
	ep = strings.TrimPrefix(ep, "http://")
	return strings.TrimRight(ep, "/")
}

func buildObjectKey(gtin string) string {
	safe := sanitizeGTIN(gtin)
	return fmt.Sprintf("products/%s/600x600.webp", safe)
}

func sanitizeGTIN(gtin string) string {
	var b strings.Builder
	for _, r := range gtin {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func localFilePath(gtin string) string {
	safe := sanitizeGTIN(gtin)
	return fmt.Sprintf("/tmp/mc-product-images/%s/image", safe)
}

// AWS Signature V4 helpers

func hashSHA256(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

func buildCanonicalHeaders(headers map[string]string, orderedKeys []string) string {
	var b strings.Builder
	for _, k := range orderedKeys {
		b.WriteString(k)
		b.WriteString(":")
		b.WriteString(headers[k])
		b.WriteString("\n")
	}
	return b.String()
}

func buildCanonicalRequest(method, uri, queryString, canonicalHeaders, signedHeaders, payloadHash string) string {
	return strings.Join([]string{method, uri, queryString, canonicalHeaders, signedHeaders, payloadHash}, "\n")
}

func buildStringToSign(amzDate, credentialScope, canonicalRequest string) string {
	requestHash := hashSHA256([]byte(canonicalRequest))
	return strings.Join([]string{"AWS4-HMAC-SHA256", amzDate, credentialScope, requestHash}, "\n")
}

func sign(secret, date, region, service, stringToSign string) string {
	kDate := hmacSHA256([]byte("AWS4"+secret), []byte(date))
	kRegion := hmacSHA256(kDate, []byte(region))
	kService := hmacSHA256(kRegion, []byte(service))
	kSigning := hmacSHA256(kService, []byte("aws4_request"))
	return hex.EncodeToString(hmacSHA256(kSigning, []byte(stringToSign)))
}

func hmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}
