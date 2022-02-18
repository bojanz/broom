// Copyright (c) 2021 Bojan Zivanovic and contributors
// SPDX-License-Identifier: Apache-2.0

package broom_test

import (
	"net/http"
	"testing"

	"github.com/bojanz/broom"
)

func TestAuthorize(t *testing.T) {
	// No credentials.
	req, _ := http.NewRequest("GET", "/test", nil)
	err := broom.Authorize(req, broom.AuthConfig{})
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}

	// Empty type.
	req, _ = http.NewRequest("GET", "/test", nil)
	err = broom.Authorize(req, broom.AuthConfig{
		Credentials: "MYKEY",
	})
	if err == nil {
		t.Error("expected Authorize() to return an error")
	}
	want := "auth type not specified"
	if err.Error() != want {
		t.Errorf("got %q, want %q", err.Error(), want)
	}

	// Invalid type.
	req, _ = http.NewRequest("GET", "/test", nil)
	err = broom.Authorize(req, broom.AuthConfig{
		Credentials: "MYKEY",
		Type:        "apikey",
	})
	if err == nil {
		t.Error("expected Authorize() to return an error")
	}
	want = `unrecognized auth type "apikey"`
	if err.Error() != want {
		t.Errorf("got %q, want %q", err.Error(), want)
	}

	// API key.
	req, _ = http.NewRequest("GET", "/test", nil)
	err = broom.Authorize(req, broom.AuthConfig{
		Credentials: "MYKEY",
		Type:        "api-key",
	})
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
	got := req.Header.Get("X-API-Key")
	want = "MYKEY"
	if got != want {
		t.Errorf(`got %q, want %q`, got, want)
	}

	// API key, custom header.
	req, _ = http.NewRequest("GET", "/test", nil)
	err = broom.Authorize(req, broom.AuthConfig{
		Credentials:  "MYKEY",
		Type:         "api-key",
		APIKeyHeader: "X-MyApp-Key",
	})
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
	got = req.Header.Get("X-MyApp-Key")
	want = "MYKEY"
	if got != want {
		t.Errorf(`got %q, want %q`, got, want)
	}

	// Basic auth.
	req, _ = http.NewRequest("GET", "/test", nil)
	err = broom.Authorize(req, broom.AuthConfig{
		Credentials: "myuser:mypass",
		Type:        "basic",
	})
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
	got = req.Header.Get("Authorization")
	want = "Basic bXl1c2VyOm15cGFzcw=="
	if got != want {
		t.Errorf(`got %q, want %q`, got, want)
	}

	// Bearer auth.
	req, _ = http.NewRequest("GET", "/test", nil)
	err = broom.Authorize(req, broom.AuthConfig{
		Credentials: "MYKEY",
		Type:        "bearer",
	})
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
	got = req.Header.Get("Authorization")
	want = "Bearer MYKEY"
	if got != want {
		t.Errorf(`got %q, want %q`, got, want)
	}
}

func TestIsJSON(t *testing.T) {
	tests := []struct {
		mediaType string
		want      bool
	}{
		{"", false},
		{"application/xml", false},
		{"application/hal+xml", false},
		{"application/x-www-form-urlencoded", false},
		{"application/json", true},
		{"application/json; charset=utf-8", true},
		{"application/vnd.api+json", true},
		{"application/vnd.api+json; charset=utf-8", true},
		{"application/hal+json", true},
		{"application/hal+json; charset=utf-8", true},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := broom.IsJSON(tt.mediaType)
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}
