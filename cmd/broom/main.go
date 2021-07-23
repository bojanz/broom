package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/bojanz/broom"
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
	usage := func() {
		fmt.Fprintln(os.Stdout, "Usage: broom PROFILE OPERATION")
		fmt.Fprintln(os.Stdout, "\nBroom is an API client powered by OpenAPI.")
		fmt.Fprintln(os.Stdout, "\nProfiles:")
		for _, profile := range cfg.Profiles() {
			fmt.Fprintln(os.Stdout, "   ", profile)
		}
	}

	// No profile specified, can't list operations.
	if len(os.Args) < 2 {
		usage()
		fmt.Fprintln(os.Stdout, "\nRun broom PROFILE to get a list of available operations.")
		return
	}
	profile := os.Args[1]
	profileCfg, ok := cfg[profile]
	if !ok {
		fmt.Fprintln(os.Stderr, "Error: unknown profile", profile)
		return
	}
	operations, err := broom.LoadOperations(profileCfg.SpecFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
	// No operation specified, list all of them.
	if len(os.Args) < 3 {
		usage()
		fmt.Fprintln(os.Stdout, "\nOperations:")
		for _, tag := range operations.Tags() {
			fmt.Fprintf(os.Stdout, "    %v\n", tag)
			for _, operation := range operations.ByTag(tag) {
				operationID := operation.ID
				if operation.Deprecated {
					operationID = fmt.Sprintf("%v (deprecated)", operationID)
				}
				fmt.Fprintf(os.Stdout, "        %v\t%v\n", operationID, operation.Summary)
			}
		}
		fmt.Fprintln(os.Stdout, "\nRun broom PROFILE OPERATION --help to view the available arguments for an operation.")
		return
	}

	operationID := os.Args[2]
	_, ok = operations.ByID(operationID)
	if !ok {
		fmt.Fprintln(os.Stderr, "Error: unknown operation", operationID)
		return
	}
}
