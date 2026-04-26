package handlers

import (
	"github.com/gin-gonic/gin"
)

// Response represents a standard API response
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Message string      `json:"message,omitempty"`
}

// SuccessResponse creates a success response
func SuccessResponse(c *gin.Context, data interface{}) {
	c.JSON(200, Response{
		Success: true,
		Data:    data,
	})
}

// CreatedResponse creates a created response
func CreatedResponse(c *gin.Context, data interface{}, message string) {
	c.JSON(201, Response{
		Success: true,
		Data:    data,
		Message: message,
	})
}

// ErrorResponse creates an error response
func ErrorResponse(c *gin.Context, status int, message string) {
	c.JSON(status, Response{
		Success: false,
		Error:   message,
	})
}

// ValidationErrorResponse creates a validation error response
func ValidationErrorResponse(c *gin.Context, errors map[string]string) {
	c.JSON(400, gin.H{
		"success": false,
		"error":   "validation failed",
		"errors":  errors,
	})
}

// NotFoundResponse creates a not found response
func NotFoundResponse(c *gin.Context, resource string) {
	c.JSON(404, Response{
		Success: false,
		Error:   resource + " not found",
	})
}

// UnauthorizedResponse creates an unauthorized response
func UnauthorizedResponse(c *gin.Context, message string) {
	if message == "" {
		message = "unauthorized"
	}
	c.JSON(401, Response{
		Success: false,
		Error:   message,
	})
}

// ForbiddenResponse creates a forbidden response
func ForbiddenResponse(c *gin.Context, message string) {
	if message == "" {
		message = "access forbidden"
	}
	c.JSON(403, Response{
		Success: false,
		Error:   message,
	})
}

// InternalErrorResponse creates an internal error response
func InternalErrorResponse(c *gin.Context) {
	c.JSON(500, Response{
		Success: false,
		Error:   "internal server error",
	})
}
