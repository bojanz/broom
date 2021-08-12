// Copyright (c) 2021 Bojan Zivanovic and contributors
// SPDX-License-Identifier: Apache-2.0

package broom_test

import (
	"testing"

	"github.com/bojanz/broom"
)

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
