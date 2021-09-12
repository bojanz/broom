// Copyright (c) 2021 Bojan Zivanovic and contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"
	"runtime"
)

// Version is the current version, replaced at build time.
var Version = "dev"

const versionDescription = `Display the Broom version`

func versionCmd(args []string) {
	fmt.Fprintf(os.Stdout, "broom %s %s/%s %s\n", Version, runtime.GOOS, runtime.GOARCH, runtime.Version())
}
