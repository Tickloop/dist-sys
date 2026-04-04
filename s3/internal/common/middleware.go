package common

import (
	"net/http"
)

type Middleware func(http.Handler) http.Handler

func Chain(hldr http.Handler, middlewares ...Middleware) http.Handler {
	for _, middleware := range middlewares {
		hldr = middleware(hldr)
	}
	return hldr
}

func MVerifyContentTypeHeader(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "application/json" {
			w.WriteHeader(http.StatusUnprocessableEntity)
			w.Write([]byte("content type header missing"))
			return
		}
		h.ServeHTTP(w, r)
	})
}

func MLogPath(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.ServeHTTP(w, r)
		w.Header().Add("x-path", r.URL.Path)
	})
}