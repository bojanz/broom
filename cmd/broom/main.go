// Copyright (c) 2021 Bojan Zivanovic and contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/fatih/color"
	flag "github.com/spf13/pflag"

	"github.com/bojanz/broom"
)

func main() {
	args := os.Args[1:]
	command := parseCommand(args)
	if command == "" {
		// No subcommand specified, print usage.
		cfg, err := broom.ReadConfig(".broom.yaml")
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			exitWithError(err)
		}
		profiles := cfg.Profiles()

		fmt.Fprintln(os.Stdout, color.YellowString("Usage:"), "broom", color.GreenString("<command>"), "[<args>]")
		fmt.Fprintln(os.Stdout, "")
		fmt.Fprintln(os.Stdout, "Broom is an API client powered by OpenAPI.")
		fmt.Fprintln(os.Stdout, "")

		w := tabwriter.NewWriter(os.Stdout, 0, 1, 4, ' ', 0)
		fmt.Fprintln(w, color.YellowString("Commands:"))
		fmt.Fprintf(w, "\t%v\t%v\n", color.GreenString("<profile>"), profileDescription)
		fmt.Fprintf(w, "\t%v\t%v\n", color.GreenString("add"), addDescription)
		fmt.Fprintf(w, "\t%v\t%v\n", color.GreenString("rm"), rmDescription)
		fmt.Fprintf(w, "\t%v\t%v\n\n", color.GreenString("version"), versionDescription)
		if len(profiles) > 0 {
			fmt.Fprintln(w, color.YellowString("Profiles:"))
			for _, profile := range profiles {
				fmt.Fprintf(w, "\t%v\n", color.GreenString(profile))
			}
		} else {
			fmt.Fprintln(w, "No profiles found. Run 'broom add' to get started.")
		}
		w.Flush()

		return
	}

	switch command {
	case "add":
		addCmd(args)
	case "rm":
		rmCmd(args)
	case "version":
		versionCmd(args)
	default:
		profileCmd(args)
	}
}

// flagUsage prints colored flag usage.
//
// Based on pflag's FlagUsagesWrapped().
// Note that support for printing default flag values was removed since Broom does not have any.
func flagUsage(fs *flag.FlagSet) {
	buf := new(bytes.Buffer)
	lines := make([]string, 0, 20)
	maxlen := 0
	flag.PrintDefaults()

	fs.VisitAll(func(f *flag.Flag) {
		if f.Hidden {
			return
		}

		line := ""
		if f.Shorthand != "" && f.ShorthandDeprecated == "" {
			line = fmt.Sprintf("    -%s, --%s", color.GreenString(f.Shorthand), color.GreenString(f.Name))
		} else {
			// An empty colored string needs to be output for the missing shorthand to allow
			// the flags to line up properly.
			line = fmt.Sprintf("        %s--%s", color.GreenString(""), color.GreenString(f.Name))
		}
		varname, usage := flag.UnquoteUsage(f)
		if varname != "" {
			line += " " + varname
		}
		if f.NoOptDefVal != "" {
			switch f.Value.Type() {
			case "string":
				line += fmt.Sprintf("[=\"%s\"]", f.NoOptDefVal)
			case "bool":
				if f.NoOptDefVal != "true" {
					line += fmt.Sprintf("[=%s]", f.NoOptDefVal)
				}
			case "count":
				if f.NoOptDefVal != "+1" {
					line += fmt.Sprintf("[=%s]", f.NoOptDefVal)
				}
			default:
				line += fmt.Sprintf("[=%s]", f.NoOptDefVal)
			}
		}

		// This special character will be replaced with spacing once the correct alignment is calculated.
		line += "\x00"
		if len(line) > maxlen {
			maxlen = len(line)
		}
		line += usage
		lines = append(lines, line)
	})

	for _, line := range lines {
		sidx := strings.Index(line, "\x00")
		spacing := strings.Repeat(" ", maxlen-sidx)
		// maxlen + 2 comes from + 1 for the \x00 and + 1 for the (deliberate) off-by-one in maxlen-sidx
		fmt.Fprintln(buf, line[:sidx], spacing, strings.Replace(line[sidx+1:], "\n", "\n"+strings.Repeat(" ", maxlen+2), -1))
	}

	fmt.Fprint(os.Stdout, buf.String())
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

// exitWithError prints the given error to stderr and exists.
func exitWithError(err error) {
	fmt.Fprintln(os.Stderr, color.RedString("Error:"), err)
	os.Exit(1)
}
