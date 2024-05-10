// main - the package where the tool begins to run
package main

import (
	"github.com/xiatechs/markdown-to-confluence/cmd"
	"os"
)

func main() {
	os.Exit(cmd.Start())
}
