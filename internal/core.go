package vre

import (
	"bufio"
	"fmt"
	"github.com/mattn/go-isatty"
	"os"
)

func read(g chan bool) {
	scanner := bufio.NewScanner(os.Stdin)

	for scanner.Scan() {
		fmt.Println(scanner.Text())
	}
	g <- true
}

// Run is the actual program entry point
func Run() {
	gate := make(chan bool)
	if !isatty.IsTerminal(os.Stdin.Fd()) {
		// piping in data
		go read(gate)
	} else {
		fmt.Println(len(os.Args))
	}

	<-gate
	fmt.Println("DONE")
}
