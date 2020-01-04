package vre

import (
	"fmt"
	"github.com/mattn/go-isatty"
	"os"
)

func Run() {
	doneChan := make(chan *Output)
	eb := NewEventBox()
	tui := NewTerminal(eb)
	reader := NewReader(eb)
	re := NewMachine(eb, doneChan)
	files := 0

	if !isatty.IsTerminal(os.Stdin.Fd()) {
		// pipe in data
		go reader.ReadFile(os.Stdin, "", true)
	} else {
		// read in files
		files = 1
		go reader.ReadFiles(os.Args[1:])
	}

	tui.Init(files)
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
					re.UpdateMachine(s)
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

		for i, d := range res.output {
			for _, line := range d {
				if files == 1 {
					os.Stdout.WriteString(re.doc[i].filename + ":")
				}
				os.Stdout.Write(*line)
				os.Stdout.WriteString("\n")
			}
		}
	}
}
