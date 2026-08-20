package httpapi

import (
	"net/http"
	"strings"
)

type originPolicy struct {
	allowed map[string]struct{}
}

func newOriginPolicy(origins []string) originPolicy {
	allowed := make(map[string]struct{}, len(origins))
	for _, value := range origins {
		origin := strings.TrimSuffix(strings.TrimSpace(value), "/")
		if origin != "" {
			allowed[origin] = struct{}{}
		}
	}
	return originPolicy{allowed: allowed}
}

func (p originPolicy) allows(origin string) bool {
	origin = strings.TrimSuffix(strings.TrimSpace(origin), "/")
	if origin == "" {
		return true
	}
	_, ok := p.allowed[origin]
	return ok
}

func (p originPolicy) allowsRequest(r *http.Request) bool {
	return p.allows(r.Header.Get("Origin"))
}

func (p originPolicy) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := strings.TrimSpace(r.Header.Get("Origin"))
		if origin != "" && !p.allows(origin) {
			writeJSON(w, http.StatusForbidden, errorResponse{Error: "Origin is not allowed.", Code: "origin_not_allowed"})
			return
		}
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Add("Vary", "Origin")
		}
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
