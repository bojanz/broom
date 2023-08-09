// Copyright (c) 2021 Bojan Zivanovic and contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/fatih/color"
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
	flags.SortFlags = false
	flags.Parse(args)

	profile := flags.Arg(0)
	cfg, err := broom.ReadConfig(".broom.yaml")
	if err != nil {
		exitWithError(err)
	}
	profileCfg, ok := cfg[profile]
	if !ok {
		exitWithError(fmt.Errorf("unknown profile %v", profile))
	}
	ops, err := broom.LoadOperations(profileCfg.SpecFile)
	if err != nil {
		exitWithError(err)
	}
	// No operation specified, list all of them.
	if flags.NArg() < 2 {
		profileUsage(profile, profileCfg.ServerURL, ops)
		return
	}

	opID := flags.Arg(1)
	op, ok := ops.ByID(opID)
	if !ok {
		exitWithError(fmt.Errorf("unknown operation %s", opID))
	}
	pathValues := flags.Args()[2:]
	if *help || len(op.Parameters.Path) > len(pathValues) || (len(op.Parameters.Body) > 0 && *body == "") {
		operationUsage(op, profile)
		flagUsage(flags)
		return
	}
	values, err := broom.ParseRequestValues(*headers, pathValues, *query, *body)
	if err != nil {
		exitWithError(err)
	}

	req, err := op.Request(profileCfg.ServerURL, values)
	if err != nil {
		exitWithError(err)
	}
	if err = broom.Authenticate(req, profileCfg.Auth); err != nil {
		exitWithError(fmt.Errorf("authenticate: %w", err))
	}
	result, err := broom.Execute(req, *verbose)
	if err != nil {
		exitWithError(err)
	}

	fmt.Fprint(os.Stdout, result.Output)
	if result.StatusCode >= http.StatusBadRequest {
		os.Exit(1)
	}
}

// profileUsage prints Broom usage for a single profile.
func profileUsage(profile string, serverURL string, ops broom.Operations) {
	fmt.Fprintln(os.Stdout, color.YellowString("Usage:"), "broom", profile, color.GreenString("<operation>"), "[--help]")
	fmt.Fprintln(os.Stdout, "")
	fmt.Fprintln(os.Stdout, "Runs the specified operation on", serverURL)
	if len(ops) > 0 {
		fmt.Fprintln(os.Stdout, "")
		fmt.Fprintln(os.Stdout, color.YellowString("Operations:"))
		w := tabwriter.NewWriter(os.Stdout, 0, 1, 4, ' ', 0)
		for _, tag := range ops.Tags() {
			fmt.Fprintf(w, "\t%v\t\t\n", tag)
			for _, op := range ops.ByTag(tag) {
				fmt.Fprintf(w, "\t    %v\t%v\n", color.GreenString(op.ID), op.SummaryWithFlags())
			}
		}
		w.Flush()
		fmt.Fprintln(os.Stdout, "")
		fmt.Fprintf(os.Stdout, "Run 'broom %v %v --help' to view the available arguments for an operation.\n", profile, color.GreenString("<operation>"))
	}
}

// operationUsage prints Broom usage for a single operation.
func operationUsage(op broom.Operation, profile string) {
	sb := strings.Builder{}
	sb.WriteString(op.ID)
	for _, param := range op.Parameters.Path {
		sb.WriteString(" " + color.GreenString("<%s>", strcase.ToSnake(param.Name)))
	}

	fmt.Fprintln(os.Stdout, color.YellowString("Usage:"), "broom", profile, sb.String())
	if summary := op.SummaryWithFlags(); summary != "" {
		fmt.Fprintln(os.Stdout, "")
		fmt.Fprintln(os.Stdout, summary)
	}
	if op.Description != "" {
		fmt.Fprintln(os.Stdout, "")
		fmt.Fprintln(os.Stdout, op.Description)
	}
	if len(op.Parameters.Header) > 0 {
		fmt.Fprintln(os.Stdout, "")
		fmt.Fprintln(os.Stdout, color.YellowString("Header parameters:"))
		w := tabwriter.NewWriter(os.Stdout, 0, 1, 4, ' ', 0)
		for _, param := range op.Parameters.Header {
			description := prepareParameterDescription(param)
			fmt.Fprintf(w, "\t%v %v\t%v\n", color.GreenString(param.Name), param.FormattedFlags(), description)
		}
		w.Flush()
	}
	if len(op.Parameters.Query) > 0 {
		fmt.Fprintln(os.Stdout, "")
		fmt.Fprintln(os.Stdout, color.YellowString("Query parameters:"))
		w := tabwriter.NewWriter(os.Stdout, 0, 1, 4, ' ', 0)
		for _, param := range op.Parameters.Query {
			description := prepareParameterDescription(param)
			fmt.Fprintf(w, "\t%v %v\t%v\n", color.GreenString(param.Name), param.FormattedFlags(), description)
		}
		w.Flush()
	}
	if len(op.Parameters.Body) > 0 {
		fmt.Fprintln(os.Stdout, "")
		fmt.Fprintln(os.Stdout, color.YellowString("Body parameters:"))
		w := tabwriter.NewWriter(os.Stdout, 0, 1, 4, ' ', 0)
		for _, param := range op.Parameters.Body {
			description := prepareParameterDescription(param)
			fmt.Fprintf(w, "\t%v %v\t%v\n", color.GreenString(param.Name), param.FormattedFlags(), description)
		}
		w.Flush()
	}
	fmt.Fprintln(os.Stdout, "")
	fmt.Fprintln(os.Stdout, color.YellowString("Options:"))
}

// prepareParameterDescription prepares a parameter description for display.
//
// If a description has multiple lines, all lines are indented to match the first line's width.
func prepareParameterDescription(p broom.Parameter) string {
	// Since colors are used for the name column, tabwriter requires color codes to
	// be present even when that column is empty, for the tab width to be right.
	return strings.ReplaceAll(p.Description, "\n", "\n\t"+color.GreenString("")+"\t")
}
