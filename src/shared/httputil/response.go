package httputil

import (
	"encoding/json"
	"math"
	"net/http"
)

type Meta struct {
	Page       int `json:"page"`
	PageSize   int `json:"page_size"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

type PaginatedResponse struct {
	Data interface{} `json:"data"`
	Meta Meta        `json:"meta"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func JSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data) //nolint:errcheck
}

func Paginated(w http.ResponseWriter, status int, data interface{}, page, pageSize, total int) {
	totalPages := 1
	if pageSize > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(pageSize)))
	}
	JSON(w, status, PaginatedResponse{
		Data: data,
		Meta: Meta{Page: page, PageSize: pageSize, Total: total, TotalPages: totalPages},
	})
}

func Error(w http.ResponseWriter, status int, msg string) {
	JSON(w, status, ErrorResponse{Error: msg})
}
