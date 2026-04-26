package handlers

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

// PaginationParams holds pagination parameters
type PaginationParams struct {
	Page    int
	Limit   int
	Offset  int
	SortBy  string
	Order   string
}

// DefaultPaginationParams returns default pagination parameters
func DefaultPaginationParams() PaginationParams {
	return PaginationParams{
		Page:   1,
		Limit:  20,
		Offset: 0,
		SortBy: "id",
		Order:  "desc",
	}
}

// GetPaginationParams extracts pagination parameters from request
func GetPaginationParams(c *gin.Context) PaginationParams {
	params := DefaultPaginationParams()

	// Parse page (default: 1)
	if pageStr := c.Query("page"); pageStr != "" {
		if page, err := strconv.Atoi(pageStr); err == nil && page > 0 {
			params.Page = page
		}
	}

	// Parse limit (default: 20, max: 100)
	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			if limit > 100 {
				limit = 100
			}
			params.Limit = limit
		}
	}

	// Calculate offset
	params.Offset = (params.Page - 1) * params.Limit

	// Parse sort by (default: id)
	if sortBy := c.Query("sort_by"); sortBy != "" {
		params.SortBy = sanitizeSortField(sortBy)
	}

	// Parse order (default: desc)
	if order := c.Query("order"); order != "" {
		order = sanitizeSortField(order)
		if order == "asc" || order == "desc" {
			params.Order = order
		}
	}

	return params
}

// PaginatedResponse represents a paginated response
type PaginatedResponse struct {
	Data       interface{} `json:"data"`
	Page       int         `json:"page"`
	Limit      int         `json:"limit"`
	TotalCount int64       `json:"total_count"`
	TotalPages int         `json:"total_pages"`
	HasNext    bool        `json:"has_next"`
	HasPrev    bool        `json:"has_prev"`
}

// NewPaginatedResponse creates a new paginated response
func NewPaginatedResponse(data interface{}, totalCount int64, params PaginationParams) PaginatedResponse {
	totalPages := int(totalCount) / params.Limit
	if int(totalCount)%params.Limit > 0 {
		totalPages++
	}

	return PaginatedResponse{
		Data:       data,
		Page:       params.Page,
		Limit:      params.Limit,
		TotalCount: totalCount,
		TotalPages: totalPages,
		HasNext:    params.Page < totalPages,
		HasPrev:    params.Page > 1,
	}
}

// sanitizeSortField prevents SQL injection in sort fields
func sanitizeSortField(field string) string {
	// Only allow alphanumeric and underscore
	allowed := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_"
	result := ""
	for _, c := range field {
		if containsRune(allowed, c) {
			result += string(c)
		}
	}
	if result == "" {
		return "id"
	}
	return result
}

func containsRune(s string, r rune) bool {
	for _, c := range s {
		if c == r {
			return true
		}
	}
	return false
}
