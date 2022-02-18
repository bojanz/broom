// Copyright (c) 2021 Bojan Zivanovic and contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	flag "github.com/spf13/pflag"

	"github.com/bojanz/broom"
)

const addDescription = `Add a new profile`

const addUsage = `Usage: broom add <profile> <spec_file>

Adds a profile to the .broom.yaml config file in the current directory.

The auth type, API key header, and server url will be auto-detected from
the specification, unless they are provided via options.

Examples:
    Single profile:
        broom add api openapi.yaml

    Single profile with Bearer auth via external command:
        broom add api openapi.json --auth-cmd="sh get-token.sh" --auth-type=bearer

    Single profile with Basic auth:
        broom add api openapi.yaml --auth="myuser:mypass" --auth-type=basic

    Multiple profiles with different API keys:
        broom add prod openapi.yaml --auth=PRODUCTION_KEY --auth-type=api-key
        broom add staging openapi.yaml --auth=STAGING_KEY --auth-type=api-key --server-url=htts://staging.my-api.io

Options:`

func addCmd(args []string) {
	authTypes := broom.AuthTypes()
	flags := flag.NewFlagSet("add", flag.ExitOnError)
	var (
		_               = flags.BoolP("help", "h", false, "Display this help text and exit")
		authCredentials = flags.String("auth", "", "Auth credentials (e.g. access token or API key). Used to authenticate every request")
		authCommand     = flags.String("auth-cmd", "", "Auth command. Executed on every request to retrieve auth credentials")
		authType        = flags.String("auth-type", "", fmt.Sprintf("Auth type. One of: %v. Defaults to %v", strings.Join(authTypes, ", "), authTypes[0]))
		apiKeyHeader    = flags.String("api-key-header", "", "API key header. Defaults to X-API-Key")
		serverURL       = flags.String("server-url", "", "Server URL")
	)
	flags.Usage = func() {
		fmt.Println(addUsage)
		flags.PrintDefaults()
	}
	flags.SortFlags = false
	flags.Parse(args)
	if flags.NArg() < 3 {
		flags.Usage()
		return
	}
	if *authType != "" && !contains(authTypes, *authType) {
		fmt.Fprintf(os.Stderr, "Error: unrecognized auth type %q\n", *authType)
		os.Exit(1)
	}

	profile := flags.Arg(1)
	filename := filepath.Clean(flags.Arg(2))
	// Ensure a profile name doesn't conflict with a command name.
	if profile == "add" || profile == "rm" || profile == "version" {
		fmt.Fprintf(os.Stderr, "Error: can't name a profile %q, please choose a different name\n", profile)
		os.Exit(1)
	}
	// Confirm that the specification exists and is valid.
	// Then use it to determine config defaults.
	spec, err := broom.LoadSpec(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	specAuthType := authTypes[0]
	specAPIKeyHeader := ""
	for _, securityScheme := range spec.Components.SecuritySchemes {
		if securityScheme.Value == nil {
			continue
		}
		if securityScheme.Value.Type == "http" && securityScheme.Value.Scheme == "bearer" {
			specAuthType = "bearer"
			break
		} else if securityScheme.Value.Type == "http" && securityScheme.Value.Scheme == "basic" {
			specAuthType = "basic"
			break
		} else if securityScheme.Value.Type == "apiKey" && securityScheme.Value.In == "header" {
			specAuthType = "api-key"
			specAPIKeyHeader = securityScheme.Value.Name
			break
		}
	}
	if *serverURL == "" && len(spec.Servers) > 0 {
		*serverURL = spec.Servers[0].URL
	}
	if *authType == "" {
		*authType = specAuthType
	}
	if *apiKeyHeader == "" {
		*apiKeyHeader = specAPIKeyHeader
	}
	profileCfg := broom.ProfileConfig{}
	profileCfg.SpecFile = filename
	profileCfg.ServerURL = *serverURL
	profileCfg.Auth = broom.AuthConfig{
		Credentials:  *authCredentials,
		Command:      *authCommand,
		Type:         *authType,
		APIKeyHeader: *apiKeyHeader,
	}

	// It is okay if the config file doesn't exist yet, so the error is ignored.
	cfg, _ := broom.ReadConfig(".broom.yaml")
	cfg[profile] = profileCfg
	if err := broom.WriteConfig(".broom.yaml", cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stdout, "Added the %v profile to .broom.yaml\n", profile)
}

// contains returns whether the sorted slice a contains x.
func contains(a []string, x string) bool {
	for _, v := range a {
		if v == x {
			return true
		}
	}
	return false
}
