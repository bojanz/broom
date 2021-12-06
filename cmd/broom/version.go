// Copyright (c) 2021 Bojan Zivanovic and contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"
	"runtime"

	"github.com/bojanz/broom"
)

const versionDescription = `Display the Broom version`

func versionCmd(args []string) {
	fmt.Fprintf(os.Stdout, "broom %s %s/%s %s\n", broom.Version, runtime.GOOS, runtime.GOARCH, runtime.Version())
}
