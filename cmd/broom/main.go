package main

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"text/tabwriter"

	flag "github.com/spf13/pflag"

	"github.com/bojanz/broom"
)

var (
	help    = flag.BoolP("help", "h", false, "Display this help text and exit")
	body    = flag.StringP("body", "b", "", "Body string, containing one or more body parameters")
	query   = flag.StringP("query", "q", "", "Query string, containing one or more query parameters")
	verbose = flag.BoolP("verbose", "v", false, "Print the HTTP status and headers hefore the response body")
)

func main() {
	cfg, err := broom.ReadConfig(".broom.yaml")
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Fprintln(os.Stderr, "Broom has not been initialized in this directory. Please run broom-init.")
		} else {
			fmt.Fprintln(os.Stderr, "Error:", err)
		}
		os.Exit(1)
	}
	flag.Usage = func() {
		flag.PrintDefaults()
	}
	flag.Parse()

	// No profile specified, can't list operations.
	if flag.NArg() < 1 {
		usage(cfg.Profiles())
		return
	}
	profile := flag.Arg(0)
	profileCfg, ok := cfg[profile]
	if !ok {
		fmt.Fprintln(os.Stderr, "Error: unknown profile", profile)
		os.Exit(1)
	}
	operations, err := broom.LoadOperations(profileCfg.SpecFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
	// No operation specified, list all of them.
	if flag.NArg() < 2 {
		profileUsage(profile, profileCfg.ServerURL, operations)
		return
	}

	operationID := flag.Arg(1)
	operation, ok := operations.ByID(operationID)
	if !ok {
		fmt.Fprintln(os.Stderr, "Error: unknown operation", operationID)
		os.Exit(1)
	}
	pathParams := operation.ParametersIn("path")
	pathValues := flag.Args()[2:]
	if *help || len(pathParams) > len(pathValues) {
		operationUsage(operation, profile)
		return
	}
	path, err := operation.RealPath(pathValues, *query)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
	bodyBytes, err := operation.ProcessBody(*body)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}

	token := profileCfg.Token
	if profileCfg.TokenCmd != "" {
		token, err = broom.RetrieveToken(profileCfg.TokenCmd)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}
	}
	req, err := http.NewRequest(operation.Method, profileCfg.ServerURL+path, bytes.NewReader(bodyBytes))
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
	if token != "" {
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %v", token))
	}
	if operation.BodyFormat != "" {
		req.Header.Add("Content-Type", operation.BodyFormat)
	}
	result, err := broom.Execute(req, *verbose)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}

	fmt.Fprint(os.Stdout, result.Output)
	if result.StatusCode >= http.StatusBadRequest {
		os.Exit(1)
	}
}

// usage prints Broom usage.
func usage(profiles []string) {
	fmt.Fprintln(os.Stdout, "Usage: broom PROFILE OPERATION")
	fmt.Fprintln(os.Stdout, "\nBroom is an API client powered by OpenAPI.")
	fmt.Fprintln(os.Stdout, "\nProfiles:")
	for _, profile := range profiles {
		fmt.Fprintln(os.Stdout, "   ", profile)
	}
	fmt.Fprintln(os.Stdout, "\nRun broom PROFILE to get a list of available operations.")
}

// usage prints Broom usage for a single profile.
func profileUsage(profile string, serverURL string, operations broom.Operations) {
	fmt.Fprintln(os.Stdout, "Usage: broom", profile, "OPERATION")
	fmt.Fprintln(os.Stdout, "\nRuns the specified operation on", serverURL)
	if len(operations) > 0 {
		fmt.Fprintln(os.Stdout, "\nOperations:")
		w := tabwriter.NewWriter(os.Stdout, 0, 1, 4, ' ', 0)
		for _, tag := range operations.Tags() {
			fmt.Fprintf(w, "\t%v\t\t\n", tag)
			for _, operation := range operations.ByTag(tag) {
				operationID := operation.ID
				if operation.Deprecated {
					operationID = fmt.Sprintf("%v (deprecated)", operationID)
				}
				fmt.Fprintf(w, "\t    %v\t%v\n", operationID, operation.Summary)
			}
		}
		w.Flush()
		fmt.Fprintln(os.Stdout, "\nRun broom PROFILE OPERATION --help to view the available arguments for an operation.")
	}
}

// operationUsage prints Broom usage for a single operation.
func operationUsage(operation broom.Operation, profile string) {
	sb := strings.Builder{}
	sb.WriteString(operation.ID)
	for _, param := range operation.ParametersIn("path") {
		sb.WriteString(" ")
		sb.WriteString(strings.ToUpper(param.Name))
	}
	summary := operation.Summary
	if summary != "" && operation.Deprecated {
		summary = fmt.Sprintf("%v (deprecated)", summary)
	}
	queryParams := operation.ParametersIn("query")
	bodyParams := operation.ParametersIn("body")

	fmt.Fprintln(os.Stdout, "Usage: broom", profile, sb.String())
	if summary != "" {
		fmt.Fprintln(os.Stdout, "")
		fmt.Fprintln(os.Stdout, summary)
	}
	if operation.Description != "" {
		fmt.Fprintln(os.Stdout, "")
		fmt.Fprintln(os.Stdout, operation.Description)
	}
	if len(queryParams) > 0 {
		fmt.Fprintln(os.Stdout, "\nQuery parameters:")
		w := tabwriter.NewWriter(os.Stdout, 0, 1, 4, ' ', 0)
		for _, param := range queryParams {
			name := param.Name
			if param.Required {
				name = fmt.Sprintf("%v (required)", name)
			} else if param.Deprecated {
				name = fmt.Sprintf("%v (deprecated)", name)
			}
			fmt.Fprintf(w, "\t%v\t%v\n", name, param.Description)
		}
		w.Flush()
	}
	if len(bodyParams) > 0 {
		fmt.Fprintln(os.Stdout, "\nBody parameters:")
		w := tabwriter.NewWriter(os.Stdout, 0, 1, 4, ' ', 0)
		for _, param := range bodyParams {
			name := param.Name
			if param.Required {
				name = fmt.Sprintf("%v (required)", name)
			} else if param.Deprecated {
				name = fmt.Sprintf("%v (deprecated)", name)
			}
			fmt.Fprintf(w, "\t%v\t%v\n", name, param.Description)
		}
		w.Flush()
	}
	fmt.Fprintln(os.Stdout, "\nOptions:")
	flag.Usage()
}
