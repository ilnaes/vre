package vre

import (
	"fmt"
	"github.com/mattn/go-isatty"
	"os"
)

func Run() {
	doneChan := make(chan []*[]byte)
	eb := NewEventBox()
	tui := NewTerminal(eb)
	reader := NewReader(eb)
	re := NewRe(eb, doneChan)

	if !isatty.IsTerminal(os.Stdin.Fd()) {
		// pipe in data
		go reader.ReadFile(os.Stdin, "", true)
	} else {
		// read in files
		go reader.ReadFiles(os.Args[1:])
	}

	tui.Init()
	go tui.Loop()
	go re.Loop()

	done := false
	early := false
	readError := ""
	for !done {
		eb.Wait(func(e *Events) {
			for eventType, v := range *e {
				switch eventType {

				case EvtSearchNew:
					s := v.(Query)
					re.UpdateRe(s)
					if len(s.input) == 0 {
						tui.ClearBounds()
					}

				case EvtSearchFinal:
					re.Finish()
					done = true

				case EvtSearchProgress:
					tui.UpdateBounds(v.(*Result))

				case EvtReadNew, EvtReadDone:
					ss := reader.Snapshot()
					tui.UpdateChunks(ss, eventType == EvtReadDone)
					re.UpdateDoc(ss, eventType == EvtReadDone)

				case EvtReadError:
					done = true
					early = true
					readError = v.(string)

				case EvtQuit:
					done = true
					early = true

				case EvtHeartbeat:
					tui.UpdatePrompt(v.(string))

				default:

				}
			}
			eb.Clear()
		})
	}

	if early {
		tui.Close()

		if readError != "" {
			fmt.Println("Problem reading " + readError)
		}
	} else {
		// print results
		res := <-doneChan
		tui.Close()

		for _, s := range res {
			os.Stdout.Write(*s)
			os.Stdout.WriteString("\n")
		}
	}
}
