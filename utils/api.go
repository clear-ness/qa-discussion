package utils

import (
	"net/http"
	"strings"
)

func CheckOrigin(r *http.Request, allowedOrigins string) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}

	if allowedOrigins == "*" {
		return true
	}
	for _, allowed := range strings.Split(allowedOrigins, " ") {
		if allowed == origin {
			return true
		}
	}

	return false
}

func OriginChecker(allowedOrigins string) func(*http.Request) bool {
	return func(r *http.Request) bool {
		return CheckOrigin(r, allowedOrigins)
	}
}
