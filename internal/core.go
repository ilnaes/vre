package vre

import (
	"fmt"
	"github.com/mattn/go-isatty"
	"os"
)

// Run is the actual program entry point
func Run() {
	t := Terminal{}
	d := Doc{}

	t.Init()
	t.GetEvent()
	t.Close()

	if !isatty.IsTerminal(os.Stdin.Fd()) {
		// piping in data
		go d.Read(os.Stdin)
	} else {
		fmt.Fprintf(os.Stdout, "%d\n", len(os.Args))
	}
}
