package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEmbeddedIconRoutes(t *testing.T) {
	mux := http.NewServeMux()
	registerIconRoutes(mux)

	for _, path := range []string{"/icon.png", "/icon_transp.png"} {
		t.Run(path, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, path, nil)
			mux.ServeHTTP(recorder, request)

			if recorder.Code != http.StatusOK {
				t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
			}
			if got := recorder.Header().Get("Content-Type"); got != "image/png" {
				t.Fatalf("expected image/png content type, got %q", got)
			}
			if got := recorder.Body.Bytes(); len(got) < 8 || string(got[:8]) != "\x89PNG\r\n\x1a\n" {
				t.Fatal("response does not contain a PNG image")
			}
		})
	}
}
