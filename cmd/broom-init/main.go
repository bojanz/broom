package main

import (
	"fmt"
	"os"
)

func main() {
	profile := "test"
	fmt.Fprintf(os.Stdout, "Initialized the %v profile in .broom.yaml\n", profile)
}
