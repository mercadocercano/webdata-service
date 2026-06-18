package image

import "testing"

func TestDecodeDataURI(t *testing.T) {
	// "hola" en base64 = aG9sYQ==
	got, err := decodeDataURI("data:image/png;base64,aG9sYQ==")
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if string(got) != "hola" {
		t.Errorf("got %q, want %q", got, "hola")
	}
}

func TestDecodeDataURI_URLSafeFallback(t *testing.T) {
	// "aG9sYQ" sin padding = RawURLEncoding de "hola"; StdEncoding falla → cae a URL-safe.
	got, err := decodeDataURI("data:image/png;base64,aG9sYQ")
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if string(got) != "hola" {
		t.Errorf("got %q, want %q", got, "hola")
	}
}

func TestDecodeDataURI_Errors(t *testing.T) {
	if _, err := decodeDataURI("data:image/png,sinbase64"); err == nil {
		t.Error("esperaba error para data URI no-base64")
	}
	if _, err := decodeDataURI("dataurisincoma"); err == nil {
		t.Error("esperaba error para URI sin coma")
	}
}
