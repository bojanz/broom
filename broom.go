// Copyright (c) 2021 Bojan Zivanovic and contributors
// SPDX-License-Identifier: Apache-2.0

package broom

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"

	strip "github.com/grokify/html-strip-tags-go"
	"github.com/tidwall/pretty"
)

// Result represents the result of executing an HTTP request.
type Result struct {
	StatusCode int
	Output     string
}

// Execute performs the given HTTP request and returns the result.
//
// The output consists of the request body (pretty-printed if JSON),
// and optionally the status code and headers (when "verbose" is true).
func Execute(req *http.Request, verbose bool) (Result, error) {
	httpClient := &http.Client{}
	resp, err := httpClient.Do(req)
	if err != nil {
		return Result{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Result{}, err
	}
	sb := strings.Builder{}
	if verbose {
		sb.WriteString(resp.Status)
		sb.WriteByte('\n')
		resp.Header.WriteSubset(&sb, nil)
		sb.WriteByte('\n')
	}
	if IsJSON(resp.Header.Get("Content-Type")) {
		body = PrettyJSON(body)
	}
	sb.Write(body)

	return Result{resp.StatusCode, sb.String()}, nil
}

// IsJSON checks whether the given media type matches a JSON format.
func IsJSON(mediaType string) bool {
	// Needs to match not just application/json, but also variants
	// such as application/vnd.api+json and application/hal+json,
	// with or without a charset suffix.
	return strings.Contains(mediaType, "json")
}

// PrettyJSON pretty-prints the given JSON.
func PrettyJSON(json []byte) []byte {
	// Many web stacks (Go, Ruby on Rails, Symfony) escape the &, <, >
	// HTML characters for safety reasons. Unescape them for readability.
	json = bytes.ReplaceAll(json, []byte("\\u0026"), []byte("&"))
	json = bytes.ReplaceAll(json, []byte("\\u003c"), []byte("<"))
	json = bytes.ReplaceAll(json, []byte("\\u003e"), []byte(">"))

	return pretty.Color(pretty.Pretty(json), nil)
}

// RetrieveToken retrieves a token by running the given command.
func RetrieveToken(tokenCmd string) (string, error) {
	errBuf := &bytes.Buffer{}
	cmd := exec.Command("sh", "-c", tokenCmd)
	cmd.Env = os.Environ()
	cmd.Stderr = errBuf
	output, err := cmd.Output()
	if err != nil {
		// The error is just a return code, which isn't useful.
		return "", fmt.Errorf("retrieve token: %v", errBuf.String())
	}
	token := strings.TrimSpace(string(output))

	return token, nil
}

// Sanitize sanitizes the given string, stripping HTML and trailing newlines.
func Sanitize(s string) string {
	return strings.Trim(strip.StripTags(s), "\n")
}

// contains returns whether slice a contains x.
func contains(a []string, x string) bool {
	for _, v := range a {
		if v == x {
			return true
		}
	}
	return false
}
