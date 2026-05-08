package service

import (
	"net/url"
	"testing"
)

func TestPasswordResetURLPlacesTokenInFragment(t *testing.T) {
	resetURL, err := passwordResetURL("https://app.example.com/reset-password?locale=ko", "raw-token")
	if err != nil {
		t.Fatalf("expected reset URL, got %v", err)
	}

	parsedURL, err := url.Parse(resetURL)
	if err != nil {
		t.Fatalf("expected valid reset URL, got %v", err)
	}

	if parsedURL.Query().Get("resetToken") != "" {
		t.Fatal("expected reset token to be absent from query")
	}
	if parsedURL.Query().Get("locale") != "ko" {
		t.Fatalf("expected existing query params to be preserved, got %q", parsedURL.RawQuery)
	}

	fragment, err := url.ParseQuery(parsedURL.Fragment)
	if err != nil {
		t.Fatalf("expected query-style reset URL fragment, got %v", err)
	}
	if fragment.Get("resetToken") != "raw-token" {
		t.Fatalf("expected reset token in fragment, got %q", parsedURL.Fragment)
	}
}
