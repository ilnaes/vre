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
	re := NewRe(eb)

	if !isatty.IsTerminal(os.Stdin.Fd()) {
		// piping in data
		go reader.Read(os.Stdin)
	} else {
		fmt.Fprintf(os.Stdout, "%d\n", len(os.Args))
	}

	tui.Init()
	go tui.Loop()
	go re.Loop()

	done := false
	for !done {
		eb.Wait(func(e *Events) {
			for eventType, v := range *e {
				switch eventType {

				case EvtSearchNew:
					re.UpdateRe(v.(string))

				case EvtSearchProgress:
					tui.UpdatePrompt(fmt.Sprintf("%v", v))
					tui.UpdateBounds(v.([][ChunkSize][][]int))

				case EvtReadNew, EvtReadDone:
					ss := reader.Snapshot()
					tui.UpdateChunks(ss)
					re.UpdateDoc(ss)

				case EvtQuit:
					done = true

				}
			}
			eb.Clear()
		})
	}
}
