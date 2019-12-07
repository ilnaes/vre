package vre

import (
	"fmt"
	"github.com/mattn/go-isatty"
	"os"
)

func Run() {
	eb := NewEventBox()
	tui := NewTerminal(eb)
	reader := NewReader(eb)

	if !isatty.IsTerminal(os.Stdin.Fd()) {
		// piping in data
		go reader.Read(os.Stdin)
	} else {
		fmt.Fprintf(os.Stdout, "%d\n", len(os.Args))
	}

	tui.Init()
	go tui.Loop()

	done := false
	for !done {
		eb.Wait(func(e *Events) {
			for eventType, _ := range *e {
				switch eventType {

				case EvtReadNew, EvtReadDone:
					ss := reader.Snapshot()
					tui.UpdateChunks(ss)

				case EvtQuit:
					done = true

				}
			}
			eb.Clear()
		})
	}
}
