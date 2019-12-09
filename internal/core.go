package vre

import (
	"fmt"
	"github.com/mattn/go-isatty"
	"os"
)

func Run() {
	doneChan := make(chan []*string)
	eb := NewEventBox()
	tui := NewTerminal(eb)
	reader := NewReader(eb)
	re := NewRe(eb, doneChan)

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
	early := false
	for !done {
		eb.Wait(func(e *Events) {
			for eventType, v := range *e {
				switch eventType {

				case EvtSearchNew:
					s := v.(string)
					re.UpdateRe(s)
					if len(s) == 0 {
						tui.ClearBounds()
					}

				case EvtSearchFinal:
					re.Finish()
					done = true

				case EvtSearchProgress:
					tui.UpdateBounds(v.([][ChunkSize][][]int))

				case EvtReadNew, EvtReadDone:
					ss := reader.Snapshot()
					tui.UpdateChunks(ss)
					re.UpdateDoc(ss, eventType == EvtReadDone)

				case EvtQuit:
					done = true
					early = true

				default:

				}
			}
			eb.Clear()
		})
	}

	if early {
		tui.Close()
	} else {
		// print results
		res := <-doneChan
		tui.Close()

		for _, s := range res {
			fmt.Fprintf(os.Stdout, *s+"\n")
		}
	}
}
