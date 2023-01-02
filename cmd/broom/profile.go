// Copyright (c) 2021 Bojan Zivanovic and contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/iancoleman/strcase"
	flag "github.com/spf13/pflag"

	"github.com/bojanz/broom"
)

const profileDescription = `List the profile's operations`

func profileCmd(args []string) {
	flags := flag.NewFlagSet("profile", flag.ExitOnError)
	var (
		help    = flags.BoolP("help", "h", false, "Display this help text and exit")
		headers = flags.StringArrayP("header", "H", nil, "Header. Can be used multiple times")
		body    = flags.StringP("body", "b", "", "Body string, containing one or more body parameters")
		query   = flags.StringP("query", "q", "", "Query string, containing one or more query parameters")
		verbose = flags.BoolP("verbose", "v", false, "Print the HTTP status and headers hefore the response body")
	)
	flags.Usage = func() {
		flags.PrintDefaults()
	}
	flags.SortFlags = false
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
	ops, err := broom.LoadOperations(profileCfg.SpecFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
	// No operation specified, list all of them.
	if flags.NArg() < 2 {
		profileUsage(profile, profileCfg.ServerURL, ops)
		return
	}

	opID := flags.Arg(1)
	op, ok := ops.ByID(opID)
	if !ok {
		fmt.Fprintln(os.Stderr, "Error: unknown operation", opID)
		os.Exit(1)
	}
	pathValues := flags.Args()[2:]
	if *help || len(op.Parameters.Path) > len(pathValues) || (len(op.Parameters.Body) > 0 && *body == "") {
		operationUsage(op, profile)
		flags.Usage()
		return
	}
	values, err := broom.ParseRequestValues(*headers, pathValues, *query, *body)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}

	req, err := op.Request(profileCfg.ServerURL, values)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
	if err = broom.Authenticate(req, profileCfg.Auth); err != nil {
		fmt.Fprintln(os.Stderr, "Error: authenticate:", err)
		os.Exit(1)
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
func profileUsage(profile string, serverURL string, ops broom.Operations) {
	fmt.Fprintln(os.Stdout, "Usage: broom", profile, "<operation>")
	fmt.Fprintln(os.Stdout, "\nRuns the specified operation on", serverURL)
	if len(ops) > 0 {
		fmt.Fprintln(os.Stdout, "\nOperations:")
		w := tabwriter.NewWriter(os.Stdout, 0, 1, 4, ' ', 0)
		for _, tag := range ops.Tags() {
			fmt.Fprintf(w, "\t%v\t\t\n", tag)
			for _, op := range ops.ByTag(tag) {
				fmt.Fprintf(w, "\t    %v\t%v\n", op.ID, op.SummaryWithFlags())
			}
		}
		w.Flush()
		fmt.Fprintf(os.Stdout, "\nRun 'broom %v <operation> --help' to view the available arguments for an operation.\n", profile)
	}
}

// operationUsage prints Broom usage for a single operation.
func operationUsage(op broom.Operation, profile string) {
	sb := strings.Builder{}
	sb.WriteString(op.ID)
	for _, param := range op.Parameters.Path {
		sb.WriteString(" <")
		sb.WriteString(strcase.ToSnake(param.Name))
		sb.WriteString(">")
	}

	fmt.Fprintln(os.Stdout, "Usage: broom", profile, sb.String())
	if summary := op.SummaryWithFlags(); summary != "" {
		fmt.Fprintln(os.Stdout, "")
		fmt.Fprintln(os.Stdout, summary)
	}
	if op.Description != "" {
		fmt.Fprintln(os.Stdout, "")
		fmt.Fprintln(os.Stdout, op.Description)
	}
	if len(op.Parameters.Header) > 0 {
		fmt.Fprintln(os.Stdout, "\nHeader parameters:")
		w := tabwriter.NewWriter(os.Stdout, 0, 1, 4, ' ', 0)
		for _, param := range op.Parameters.Header {
			description := strings.ReplaceAll(param.Description, "\n", "\n\t\t")
			fmt.Fprintf(w, "\t%v\t%v\n", param.NameWithFlags(), description)
		}
		w.Flush()
	}
	if len(op.Parameters.Query) > 0 {
		fmt.Fprintln(os.Stdout, "\nQuery parameters:")
		w := tabwriter.NewWriter(os.Stdout, 0, 1, 4, ' ', 0)
		for _, param := range op.Parameters.Query {
			description := strings.ReplaceAll(param.Description, "\n", "\n\t\t")
			fmt.Fprintf(w, "\t%v\t%v\n", param.NameWithFlags(), description)
		}
		w.Flush()
	}
	if len(op.Parameters.Body) > 0 {
		fmt.Fprintln(os.Stdout, "\nBody parameters:")
		w := tabwriter.NewWriter(os.Stdout, 0, 1, 4, ' ', 0)
		for _, param := range op.Parameters.Body {
			description := strings.ReplaceAll(param.Description, "\n", "\n\t\t")
			fmt.Fprintf(w, "\t%v\t%v\n", param.NameWithFlags(), description)
		}
		w.Flush()
	}
	fmt.Fprintln(os.Stdout, "\nOptions:")
}
