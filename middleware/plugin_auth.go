package middleware

import (
	"net/http"
	"strings"

	ctx "github.com/gophish/gophish/context"
	"github.com/gophish/gophish/models"
)

// RequirePluginAPIKey authenticates requests from the report button plugin
// using a per-org plugin API key passed as a Bearer token. On success it
// stores the ReportButtonConfig in the request context under "plugin_config".
func RequirePluginAPIKey(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var apiKey string
		if tokens, ok := r.Header["Authorization"]; ok && len(tokens) >= 1 {
			apiKey = strings.TrimPrefix(tokens[0], "Bearer ")
		}
		if apiKey == "" {
			JSONError(w, http.StatusUnauthorized, "Plugin API key not provided")
			return
		}
		config, err := models.GetReportButtonConfigByAPIKey(apiKey)
		if err != nil {
			JSONError(w, http.StatusUnauthorized, "Invalid plugin API key")
			return
		}
		r = ctx.Set(r, "plugin_config", config)
		next.ServeHTTP(w, r)
	})
}
