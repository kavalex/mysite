package middleware

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
)

type cspNonceKey struct{}

// CSPNonceFrom returns the per-request nonce for inline <script>/<style>.
func CSPNonceFrom(ctx context.Context) string {
	if v, ok := ctx.Value(cspNonceKey{}).(string); ok {
		return v
	}
	return ""
}

func generateCSPNonce() string {
	b := make([]byte, 16)
	rand.Read(b) // crypto/rand does not return an error since Go 1.26
	return base64.StdEncoding.EncodeToString(b)
}

func SecurityHeaders(env string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Nonce is generated before setting the CSP header.
			nonce := generateCSPNonce()
			ctx := context.WithValue(r.Context(), cspNonceKey{}, nonce)

			h := w.Header()
			csp := fmt.Sprintf(
				"default-src 'self'; "+
					"script-src 'self' 'nonce-%s'; "+
					"style-src 'self' 'nonce-%s'; "+
					"img-src 'self' data:; "+
					"font-src 'self'; "+
					"connect-src 'self'; "+
					"frame-ancestors 'none'; "+
					"base-uri 'self'; "+
					"form-action 'self'",
				nonce, nonce)
			h.Set("Content-Security-Policy", csp)

			h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
			h.Set("Permissions-Policy", "geolocation=(), camera=(), microphone=(), payment=()")
			h.Set("Cross-Origin-Opener-Policy", "same-origin")

			if env == "prod" {
				h.Set("X-Frame-Options", "DENY")
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
