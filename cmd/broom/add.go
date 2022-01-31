// Copyright (c) 2021 Bojan Zivanovic and contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"
	"path/filepath"

	flag "github.com/spf13/pflag"

	"github.com/bojanz/broom"
)

const addDescription = `Add a new profile`

const addUsage = `Usage: broom add <profile> <spec_file>

Adds a profile to the .broom.yaml config file in the current directory.

Examples:
    Single profile:
        broom add api openapi.yaml

    Multiple profiles and an API key:
        broom add prod openapi.json --token=PRODUCTION_KEY
        broom add staging openapi.json --token=STAGING_KEY --server-url=htts://staging.my-api.io

    Authentication through an external command (e.g. for OAuth):
        broom add api openapi.json --token-cmd="sh get-token.sh"

Options:`

func addCmd(args []string) {
	flags := flag.NewFlagSet("add", flag.ExitOnError)
	var (
		_         = flags.BoolP("help", "h", false, "Display this help text and exit")
		serverURL = flags.String("server-url", "", "Server URL. Overrides the one from the specification file")
		token     = flags.String("token", "", "Access token. Used to authorize every request")
		tokenCmd  = flags.String("token-cmd", "", "Access token command. Executed on every request to retrieve a token")
	)
	flags.Usage = func() {
		fmt.Println(addUsage)
		flags.PrintDefaults()
	}
	flags.Parse(args)
	if flags.NArg() < 3 {
		flags.Usage()
		return
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
	var specServerURL string
	if len(spec.Servers) > 0 {
		specServerURL = spec.Servers[0].URL
	}
	if *serverURL == "" {
		*serverURL = specServerURL
	}
	profileCfg := broom.ProfileConfig{}
	profileCfg.SpecFile = filename
	profileCfg.ServerURL = *serverURL
	profileCfg.Token = *token
	profileCfg.TokenCmd = *tokenCmd

	// It is okay if the config file doesn't exist yet, so the error is ignored.
	cfg, _ := broom.ReadConfig(".broom.yaml")
	cfg[profile] = profileCfg
	if err := broom.WriteConfig(".broom.yaml", cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stdout, "Added the %v profile to .broom.yaml\n", profile)
}
