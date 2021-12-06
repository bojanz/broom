// Copyright (c) 2021 Bojan Zivanovic and contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/iancoleman/strcase"
	"github.com/rivo/tview"
	flag "github.com/spf13/pflag"

	"github.com/bojanz/broom"
)

const profileDescription = `List the profile's operations`

func profileCmd(args []string) {
	flags := flag.NewFlagSet("profile", flag.ExitOnError)
	var (
		headers = flags.StringArrayP("header", "H", nil, "Header. Can be used multiple times")
		help    = flags.BoolP("help", "h", false, "Display this help text and exit")
		body    = flags.StringP("body", "b", "", "Body string, containing one or more body parameters")
		query   = flags.StringP("query", "q", "", "Query string, containing one or more query parameters")
		token   = flags.StringP("token", "t", "", "Access token. Overrides the one from the profile")
		verbose = flags.BoolP("verbose", "v", false, "Print the HTTP status and headers hefore the response body")
	)
	flags.Usage = func() {
		flags.PrintDefaults()
	}
	flags.Parse(args)

	profile := flags.Arg(0)
	cfg, err := broom.ReadConfig(".broom.yaml")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
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
	if flags.NArg() < 2 {
		profileUsage(profile, profileCfg.ServerURL, operations)
		return
	}

	operationID := flags.Arg(1)
	operation, ok := operations.ByID(operationID)
	if !ok {
		fmt.Fprintln(os.Stderr, "Error: unknown operation", operationID)
		os.Exit(1)
	}
	pathParams := operation.ParametersIn("path")
	pathValues := flags.Args()[2:]
	if *help || len(pathParams) > len(pathValues) {
		operationUsage(operation, profile)
		flags.Usage()
		return
	}
	path, err := operation.RealPath(pathValues, *query)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
	// The operation has a body, but no body string was provided.
	// Launch the terminal UI to collect values.
	if *body == "" && operation.HasBody() {
		var canceled bool
		*body, canceled = bodyForm(operation)
		if canceled {
			os.Exit(0)
		}
	}
	bodyBytes, err := operation.ProcessBody(*body)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
	if *token == "" {
		// No access token specified, take it from the profile.
		*token = profileCfg.Token
		if profileCfg.TokenCmd != "" {
			*token, err = broom.RetrieveToken(profileCfg.TokenCmd)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}
		}
	}

	req, err := http.NewRequest(operation.Method, profileCfg.ServerURL+path, bytes.NewReader(bodyBytes))
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
	if operation.HasBody() {
		req.Header.Set("Content-Type", operation.BodyFormat)
	}
	if *token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", *token))
	}
	for _, header := range *headers {
		kv := strings.SplitN(header, ":", 2)
		req.Header.Set(strings.TrimSpace(kv[0]), strings.TrimSpace(kv[1]))
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

// profileUsage prints Broom usage for a single profile.
func profileUsage(profile string, serverURL string, operations broom.Operations) {
	fmt.Fprintln(os.Stdout, "Usage: broom", profile, "<operation>")
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
		fmt.Fprintf(os.Stdout, "\nRun 'broom %v <operation> --help' to view the available arguments for an operation.\n", profile)
	}
}

// operationUsage prints Broom usage for a single operation.
func operationUsage(operation broom.Operation, profile string) {
	sb := strings.Builder{}
	sb.WriteString(operation.ID)
	for _, param := range operation.ParametersIn("path") {
		sb.WriteString(" <")
		sb.WriteString(strcase.ToSnake(param.Name))
		sb.WriteString(">")
	}
	summary := operation.Summary
	if summary != "" && operation.Deprecated {
		summary = fmt.Sprintf("%v (deprecated)", summary)
	}
	queryParams := operation.ParametersIn("query")
	headerParams := operation.ParametersIn("header")
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
			fmt.Fprintf(w, "\t%v\t%v\n", param.NameWithFlags(), param.Description)
		}
		w.Flush()
	}
	if len(headerParams) > 0 {
		fmt.Fprintln(os.Stdout, "\nHeader parameters:")
		w := tabwriter.NewWriter(os.Stdout, 0, 1, 4, ' ', 0)
		for _, param := range headerParams {
			fmt.Fprintf(w, "\t%v\t%v\n", param.NameWithFlags(), param.Description)
		}
		w.Flush()
	}
	if len(bodyParams) > 0 {
		fmt.Fprintln(os.Stdout, "\nBody parameters:")
		w := tabwriter.NewWriter(os.Stdout, 0, 1, 4, ' ', 0)
		for _, param := range bodyParams {
			fmt.Fprintf(w, "\t%v\t%v\n", param.NameWithFlags(), param.Description)
		}
		w.Flush()
	}
	fmt.Fprintln(os.Stdout, "\nOptions:")
}

// bodyForm renders a form for entering body parameters.
func bodyForm(operation broom.Operation) (string, bool) {
	values := url.Values{}
	canceled := false
	app := tview.NewApplication()
	form := tview.NewForm()
	cancelFunc := func() {
		values = url.Values{}
		canceled = true
		app.Stop()
	}
	form.SetCancelFunc(cancelFunc)
	form.SetBorder(true)
	form.SetTitle(operation.Summary)
	form.SetTitleAlign(tview.AlignLeft)

	bodyParams := operation.ParametersIn("body")
	for _, param := range bodyParams {
		paramName := param.Name
		paramLabel := param.Label()
		if param.Required {
			paramLabel = fmt.Sprintf("%v*", paramLabel)
		}
		paramDefault := ""
		if param.Default != nil {
			paramDefault = fmt.Sprintf("%v", param.Default)
		}

		if param.Type == "boolean" {
			form.AddCheckbox(paramLabel, paramDefault == "true", func(checked bool) {
				values.Set(paramName, strconv.FormatBool(checked))
			})
		} else if len(param.Enum) > 0 {
			initialOption := 0
			for k, v := range param.Enum {
				if v == paramDefault {
					initialOption = k
					break
				}
			}

			form.AddDropDown(paramLabel, param.Enum, initialOption, func(option string, optionIndex int) {
				values.Set(paramName, option)
			})
		} else {
			form.AddInputField(paramLabel, paramDefault, 40, nil, func(text string) {
				values.Set(paramName, text)
			})
		}
	}
	form.AddButton("Submit", func() {
		// Allow submit only if the input is valid.
		err := bodyParams.Validate(values)
		if err == nil {
			app.Stop()
		}
	})
	form.AddButton("Cancel", cancelFunc)

	if err := app.SetRoot(form, true).EnableMouse(true).Run(); err != nil {
		panic(err)
	}

	return values.Encode(), canceled
}
