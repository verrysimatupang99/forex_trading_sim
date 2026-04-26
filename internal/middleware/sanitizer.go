package middleware

import (
	"html"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
)

// InputSanitizer sanitizes user input to prevent XSS and injection attacks
func InputSanitizer() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Sanitize query parameters
		for key, values := range c.Request.URL.Query() {
			for i, value := range values {
				c.Request.URL.Query()[key][i] = sanitize(value)
			}
		}

		// Sanitize form parameters
		c.Request.PostForm = sanitizeMap(c.Request.PostForm)

		// Sanitize JSON body would require re-reading body
		// For now, we'll handle this at handler level

		c.Next()
	}
}

// sanitize removes potentially dangerous characters
func sanitize(input string) string {
	// HTML escape to prevent XSS
	input = html.EscapeString(input)

	// Remove null bytes
	input = strings.ReplaceAll(input, "\x00", "")

	// Trim whitespace
	input = strings.TrimSpace(input)

	return input
}

// sanitizeMap sanitizes all values in a map
func sanitizeMap(m map[string][]string) map[string][]string {
	sanitized := make(map[string][]string)
	for key, values := range m {
		sanitized[key] = make([]string, len(values))
		for i, value := range values {
			sanitized[key][i] = sanitize(value)
		}
	}
	return sanitized
}

// SQLInjectionMatcher detects potential SQL injection patterns
var sqlInjectionPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(union\s+select|select\s+.*\s+from|insert\s+into|delete\s+from|drop\s+table|update\s+.*\s+set)`),
	regexp.MustCompile(`(?i)(or\s+1\s*=\s*1|and\s+1\s*=\s*1|'|";|--|\/\*)`),
	regexp.MustCompile(`(?i)(exec\s*\(|execute\s*\(|xp_)`),
}

// DetectSQLInjection checks if input contains potential SQL injection
func DetectSQLInjection(input string) bool {
	for _, pattern := range sqlInjectionPatterns {
		if pattern.MatchString(input) {
			return true
		}
	}
	return false
}

// SQLInjectionMiddleware detects and blocks SQL injection attempts
func SQLInjectionMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check query parameters
		for _, values := range c.Request.URL.Query() {
			for _, value := range values {
				if DetectSQLInjection(value) {
					c.JSON(400, gin.H{
						"error": "potentially malicious input detected",
					})
					c.Abort()
					return
				}
			}
		}

		// Check form parameters
		for _, values := range c.Request.PostForm {
			for _, value := range values {
				if DetectSQLInjection(value) {
					c.JSON(400, gin.H{
						"error": "potentially malicious input detected",
					})
					c.Abort()
					return
				}
			}
		}

		c.Next()
	}
}
