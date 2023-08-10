// Copyright (c) 2021 Bojan Zivanovic and contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"

	"github.com/fatih/color"
	flag "github.com/spf13/pflag"

	"github.com/bojanz/broom"
)

const rmDescription = `Remove a profile`

func rmCmd(args []string) {
	flags := flag.NewFlagSet("rm", flag.ExitOnError)
	help := flags.BoolP("help", "h", false, "Display this help text and exit")
	flags.SortFlags = false
	flags.Parse(args)
	if *help || flags.NArg() < 2 {
		rmUsage()
		flagUsage(flags)
		return
	}

	profile := flags.Arg(1)
	cfg, err := broom.ReadConfig(".broom.yaml")
	if err != nil {
		exitWithError(err)
	}
	delete(cfg, profile)
	if err := broom.WriteConfig(".broom.yaml", cfg); err != nil {
		exitWithError(err)
	}
	fmt.Fprintf(color.Output, "Removed the %v profile from .broom.yaml\n", profile)
}

func rmUsage() {
	fmt.Fprintln(color.Output, color.YellowString("Usage:"), "broom rm", color.GreenString("<profile>"))
	fmt.Fprintln(color.Output, "")
	fmt.Fprintln(color.Output, "Removes a profile from the .broom.yaml config file in the current directory.")
	fmt.Fprintln(color.Output, "")
	fmt.Fprintln(color.Output, color.YellowString("Options:"))
}
