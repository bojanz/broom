package broom

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

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
	httpClient := &http.Client{
		Timeout: 5 * time.Second,
	}
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
	if IsJSON(resp) {
		body = PrettyJSON(body)
	}
	sb.Write(body)

	return Result{resp.StatusCode, sb.String()}, nil
}

// Checks whether the given response is in JSON format.
func IsJSON(resp *http.Response) bool {
	contentType := resp.Header.Get("Content-Type")
	// An imprecise check allows for both application/json
	// and other mime types like JSON API or HAL.
	return strings.Contains(contentType, "json")
}

// PrettyJSON pretty-prints the given JSON.
func PrettyJSON(json []byte) []byte {
	return pretty.Color(pretty.Pretty(json), nil)
}

// RetrieveToken retrieves a token by running the given command.
func RetrieveToken(tokenCmd string) (string, error) {
	cmd := exec.Command("sh", "-c", tokenCmd)
	cmd.Env = os.Environ()
	output, err := cmd.CombinedOutput()
	if err != nil {
		// The error is just a return code, which isn't useful.
		return "", fmt.Errorf("retrieve token: %v", string(output))
	}
	token := strings.TrimSpace(string(output))

	return token, nil
}
