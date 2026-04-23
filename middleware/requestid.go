package middleware

import (
	"net/http"

	ctx "github.com/gophish/gophish/context"
	"github.com/google/uuid"
)

// RequestID ensures every request has an X-Request-ID header for log
// correlation. If the incoming request already carries the header its value is
// preserved; otherwise a new UUID is generated. The ID is set on the response
// and stored in the request context under the key "request_id".
func RequestID(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = uuid.New().String()
		}
		w.Header().Set("X-Request-ID", id)
		r = ctx.Set(r, "request_id", id)
		next.ServeHTTP(w, r)
	}
}
