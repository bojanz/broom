// Copyright (c) 2021 Bojan Zivanovic and contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/bojanz/broom"
)

func main() {
	args := os.Args[1:]
	command := parseCommand(args)
	if command == "" {
		// No subcommand specified, print usage.
		cfg, err := broom.ReadConfig(".broom.yaml")
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}
		profiles := cfg.Profiles()

		fmt.Fprintln(os.Stdout, "Usage: broom [--help] <command> [<args>]")
		fmt.Fprintln(os.Stdout, "\nBroom is an API client powered by OpenAPI.")

		w := tabwriter.NewWriter(os.Stdout, 0, 1, 4, ' ', 0)
		fmt.Fprintln(w, "\nCommands:")
		fmt.Fprintf(w, "\t<profile>\t%v\n", profileDescription)
		fmt.Fprintf(w, "\tinit\t%v\n", initDescription)
		if len(profiles) > 0 {
			fmt.Fprintln(w, "\nProfiles:")
			for _, profile := range profiles {
				fmt.Fprintf(w, "\t%v\n", profile)
			}
		} else {
			fmt.Fprintln(w, "\nNo profiles found. Run 'broom init' to get started.")
		}
		w.Flush()

		return
	}

	switch command {
	case "init":
		initCmd(args)
	default:
		profileCmd(args)
	}
}

// parseCommand returns the requested command, ignoring unparsed flags.
func parseCommand(args []string) string {
	command := ""
	for _, arg := range args {
		if !strings.HasPrefix(arg, "-") {
			command = arg
			break
		}
	}
	return command
}
