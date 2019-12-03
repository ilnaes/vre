package vre

import (
	"bufio"
	"fmt"
	"github.com/mattn/go-isatty"
	"os"
)

func read() {
	scanner := bufio.NewScanner(os.Stdin)

	for scanner.Scan() {
		fmt.Fprintf(os.Stdout, scanner.Text()+"\n")
	}
}

// Run is the actual program entry point
func Run() {
	t := Terminal{}

	t.Init()
	t.GetEvent()
	t.Close()

	if !isatty.IsTerminal(os.Stdin.Fd()) {
		// piping in data
		read()
	} else {
		fmt.Fprintf(os.Stdout, "%d\n", len(os.Args))
	}
}
