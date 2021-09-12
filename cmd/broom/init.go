// Copyright (c) 2021 Bojan Zivanovic and contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"
	"path/filepath"

	flag "github.com/spf13/pflag"
	"gopkg.in/yaml.v2"

	"github.com/bojanz/broom"
)

const initDescription = `Create a new profile`

const initUsage = `Usage: broom init <profile> <spec_file>

Creates a Broom profile using the given OpenAPI specification.

The profile is added to a .broom.yaml config file in the current directory.

Examples:
    Single profile:
        broom init api openapi.yaml

    Multiple profiles and an API key:
        broom init prod openapi.json --token=PRODUCTION_KEY
        broom init staging openapi.json --token=STAGING_KEY --server-url=htts://staging.my-api.io

    Authentication through an external command (e.g. for OAuth):
        broom init api openapi.json --token-cmd="sh get-token.sh"

Options:`

func initCmd(args []string) {
	flags := flag.NewFlagSet("init", flag.ExitOnError)
	var (
		_         = flags.BoolP("help", "h", false, "Display this help text and exit")
		serverURL = flags.String("server-url", "", "Server URL. Overrides the one from the specification file")
		token     = flags.String("token", "", "Access token. Used to authorize every request")
		tokenCmd  = flags.String("token-cmd", "", "Access token command. Executed on every request to retrieve a token")
	)
	flags.Usage = func() {
		fmt.Println(initUsage)
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
	if profile == "init" {
		fmt.Fprintf(os.Stderr, "Error: can't name a profile %q, please choose a different name\n", profile)
		os.Exit(1)
	}
	// Confirm that the spec file exists and contains valid JSON or YAML.
	b, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	aux := struct {
		Servers []struct {
			URL string `yaml:"url"`
		} `yaml:"servers"`
	}{}
	if err := yaml.Unmarshal(b, &aux); err != nil {
		fmt.Fprintf(os.Stderr, "Error: could not decode %v: %v\n", filename, err)
		os.Exit(1)
	}
	var specServerURL string
	if len(aux.Servers) > 0 {
		specServerURL = aux.Servers[0].URL
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
	fmt.Fprintf(os.Stdout, "Initialized the %v profile in .broom.yaml\n", profile)
}
