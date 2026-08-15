package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"superdaemon/config"
)

func TestSetAccessControlHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	for name, tt := range map[string]struct {
		allowedOrigins     []string
		origin             string
		wantCredentials    string
		wantAllowOrigin    string
	}{
		"normal config allows credentials": {
			allowedOrigins:  []string{"https://panel.example.com"},
			origin:          "https://panel.example.com",
			wantCredentials: "true",
			wantAllowOrigin: "https://panel.example.com",
		},
		"wildcard config disables credentials": {
			allowedOrigins:  []string{"*"},
			origin:          "https://evil.example.com",
			wantCredentials: "false",
			wantAllowOrigin: "*",
		},
	} {
		t.Run(name, func(t *testing.T) {
			// Reset global config to a known good state after each subtest.
			defer config.Set(&config.Configuration{AuthenticationToken: "reset-token", Token: config.Token{Token: "reset-token"}})

			config.Set(&config.Configuration{
				AuthenticationToken: "test-token",
				Token:               config.Token{Token: "test-token"},
				PanelLocation:       "https://panel.example.com",
				AllowedOrigins:      tt.allowedOrigins,
			})

			router := gin.New()
			router.Use(SetAccessControlHeaders())
			router.GET("/test", func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("Origin", tt.origin)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code)
			assert.Equal(t, tt.wantCredentials, rec.Header().Get("Access-Control-Allow-Credentials"))
			assert.Equal(t, tt.wantAllowOrigin, rec.Header().Get("Access-Control-Allow-Origin"))
		})
	}
}
