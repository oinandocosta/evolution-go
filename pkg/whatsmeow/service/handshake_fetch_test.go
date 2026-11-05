package whatsmeow_service

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type versionTransport func(*http.Request) (*http.Response, error)

func (transport versionTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return transport(request)
}

func TestHandshakeVersionFetchCache(t *testing.T) {
	previousTransport := http.DefaultTransport
	previousVersion, previousTime := cachedWebVersion, cachedWebVersionAt
	t.Cleanup(func() {
		http.DefaultTransport = previousTransport
		cachedWebVersion, cachedWebVersionAt = previousVersion, previousTime
	})
	cachedWebVersion, cachedWebVersionAt = nil, time.Time{}
	requests := 0
	http.DefaultTransport = versionTransport(func(request *http.Request) (*http.Response, error) {
		requests++
		deadline, bounded := request.Context().Deadline()
		if !bounded || time.Until(deadline) > 10*time.Second {
			t.Error("version lookup has no bounded deadline")
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"client_revision":1053085257}`)), Header: make(http.Header)}, nil
	})
	first, err := fetchWhatsAppWebVersion()
	if err != nil || first == nil || first.Patch != 1053085257 {
		t.Fatalf("unexpected web version: %+v, %v", first, err)
	}
	if _, err := fetchWhatsAppWebVersion(); err != nil || requests != 1 {
		t.Fatalf("cached lookup repeated the HTTP request: count=%d, error=%v", requests, err)
	}
	invalidateCachedWhatsAppWebVersion()
	if _, err := fetchWhatsAppWebVersion(); err != nil || requests != 2 {
		t.Fatalf("outdated client did not refresh the cached revision: count=%d, error=%v", requests, err)
	}
	invalidateCachedWhatsAppWebVersion()
	http.DefaultTransport = versionTransport(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusServiceUnavailable, Body: io.NopCloser(strings.NewReader(`{"client_revision":1}`)), Header: make(http.Header)}, nil
	})
	if _, err := fetchWhatsAppWebVersion(); err == nil {
		t.Fatal("non-success HTTP response was treated as a valid version")
	}
}
