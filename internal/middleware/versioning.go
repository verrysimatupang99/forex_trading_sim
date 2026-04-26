package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// APIVersion represents API version information
type APIVersion struct {
	Major     int
	Minor     int
	Raw       string
	Deprecated bool
}

var (
	// Supported versions
	supportedVersions = []APIVersion{
		{Major: 1, Minor: 0, Raw: "v1.0"},
		{Major: 1, Minor: 1, Raw: "v1.1"},
	}

	// Current stable version
	currentVersion = APIVersion{Major: 1, Minor: 1, Raw: "v1.1"}
)

// VersionHandler handles API versioning
func VersionHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get URL path for version extraction
		path := c.Request.URL.Path
		
		// Extract version from URL path (e.g., /api/v1.0/...)
		_ = extractVersionFromPath(path)
		
		// Set version in context
		c.Set("api_version", currentVersion)
		c.Set("api_version_raw", currentVersion.Raw)
		
		// Add version headers to response
		c.Header("X-API-Version", currentVersion.Raw)
		c.Header("X-API-Supported-Versions", getSupportedVersionsString())
		
		c.Next()
	}
}

// extractVersionFromPath extracts version from URL path
func extractVersionFromPath(path string) string {
	// Match /api/v1.x/ pattern
	parts := strings.Split(path, "/")
	for i, part := range parts {
		if part == "api" && i+1 < len(parts) {
			versionPart := parts[i+1]
			if strings.HasPrefix(versionPart, "v") {
				return versionPart
			}
		}
	}
	return currentVersion.Raw
}

// getSupportedVersionsString returns comma-separated supported versions
func getSupportedVersionsString() string {
	versions := make([]string, len(supportedVersions))
	for i, v := range supportedVersions {
		versions[i] = v.Raw
	}
	return strings.Join(versions, ", ")
}

// VersionMiddleware creates version-specific middleware
func VersionMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		version := c.GetString("api_version_raw")
		
		// Check if version is deprecated
		for _, v := range supportedVersions {
			if v.Raw == version && v.Deprecated {
				c.Header("X-API-Deprecation", "true")
				c.Header("X-API-Deprecation-Date", "2025-12-31")
			}
		}
		
		c.Next()
	}
}

// RequireVersion creates middleware that requires specific version
func RequireVersion(requiredVersion string) gin.HandlerFunc {
	return func(c *gin.Context) {
		version := c.GetString("api_version_raw")
		
		if version != requiredVersion {
			c.JSON(http.StatusPreconditionFailed, gin.H{
				"error":                  "version mismatch",
				"required_version":        requiredVersion,
				"current_version":        version,
				"supported_versions":      getSupportedVersionsString(),
				"upgrade_instructions":    "Please update your API client to use the latest version",
			})
			c.Abort()
			return
		}
		
		c.Next()
	}
}
