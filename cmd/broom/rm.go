// Copyright (c) 2021 Bojan Zivanovic and contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"

	flag "github.com/spf13/pflag"

	"github.com/bojanz/broom"
)

const rmDescription = `Remove a profile`

const rmUsage = `Usage: broom rm <profile>

Removes a profile from the .broom.yaml config file in the current directory.

Options:`

func rmCmd(args []string) {
	flags := flag.NewFlagSet("rm", flag.ExitOnError)
	flags.BoolP("help", "h", false, "Display this help text and exit")
	flags.Usage = func() {
		fmt.Println(rmUsage)
		flags.PrintDefaults()
	}
	flags.Parse(args)
	if flags.NArg() < 2 {
		flags.Usage()
		return
	}

	profile := flags.Arg(1)
	cfg, err := broom.ReadConfig(".broom.yaml")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	delete(cfg, profile)
	if err := broom.WriteConfig(".broom.yaml", cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stdout, "Removed the %v profile from .broom.yaml\n", profile)
}
