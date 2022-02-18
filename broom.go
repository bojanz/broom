// Copyright (c) 2021 Bojan Zivanovic and contributors
// SPDX-License-Identifier: Apache-2.0

package broom

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"

	strip "github.com/grokify/html-strip-tags-go"
	"github.com/tidwall/pretty"
)

// Version is the current version, replaced at build time.
var Version = "dev"

// Result represents the result of executing an HTTP request.
type Result struct {
	StatusCode int
	Output     string
}

// Authorize acquires access credentials and sets them on the request.
func Authorize(req *http.Request, cfg AuthConfig) error {
	if cfg.Credentials == "" && cfg.Command == "" {
		return nil
	}
	credentials := cfg.Credentials
	if cfg.Command != "" {
		var err error
		credentials, err = RunCommand(cfg.Command)
		if err != nil {
			return fmt.Errorf("run command: %w", err)
		}
		if credentials == "" {
			return fmt.Errorf("run command: no credentials received")
		}
	}

	switch cfg.Type {
	case "bearer":
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", credentials))
	case "basic":
		credentials = base64.StdEncoding.EncodeToString([]byte(credentials))
		req.Header.Set("Authorization", fmt.Sprintf("Basic %v", credentials))
	case "api-key":
		key := cfg.APIKeyHeader
		if key == "" {
			key = "X-API-Key"
		}
		req.Header.Set(key, credentials)
	case "":
		return errors.New("auth type not specified")
	default:
		return fmt.Errorf("unrecognized auth type %q", cfg.Type)
	}

	return nil
}

// AuthTypes returns a list of supported auth types.
func AuthTypes() []string {
	return []string{"bearer", "basic", "api-key"}
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

// RunCommand runs the given command and returns its output.
//
// The command has access to environment variables.
func RunCommand(command string) (string, error) {
	errBuf := &bytes.Buffer{}
	cmd := exec.Command("sh", "-c", command)
	cmd.Env = os.Environ()
	cmd.Stderr = errBuf
	b, err := cmd.Output()
	if err != nil {
		// The error is just a return code, which isn't useful.
		return "", errors.New(errBuf.String())
	}
	output := bytes.TrimSpace(b)

	return string(output), nil
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
