package main

import (
	"fmt"
	"github.com/kokizzu/goproc"
	"sync"
)

func main() {
	wg := sync.WaitGroup{}

	runner := goproc.New()

	// callback API demo
	cmdId := runner.AddCommand(&goproc.Cmd{
		Program:    `sleep`,
		Parameters: []string{`2`},
		MaxRestart: goproc.RestartForever,
		OnStderr: func(cmd *goproc.Cmd, s string) error {
			fmt.Println(`OnStderr0: ` + s)
			return nil
		},
		OnStdout: func(cmd *goproc.Cmd, s string) error {
			fmt.Println(`OnStdout0: ` + s)
			return nil
		},
		OnExit: func(cmd *goproc.Cmd) {
			wg.Done()
			return
		},
	})

	// channel API demo
	cmd := &goproc.Cmd{
		Program:        `ps`,
		Parameters:     []string{`ux`},
		RestartDelayMs: 3000,
		MaxRestart:     2,
		UseChannelApi:  true, // note: must handle all 4 channel, or it will stuck
		// callback API also still can be used
	}
	cmd2id := runner.AddCommand(cmd)

	go (func() {
		for {
			select {
			case line := <-cmd.StdoutChannel:
				fmt.Println(`OnStderr1: ` + line)
			case line := <-cmd.StderrChannel:
				fmt.Println(`OnStderr1: ` + line)
			case <-cmd.ProcesssCompletedChannel:
				// do nothing, but must be implemented or the channel will stuck
			case <-cmd.ExitChannel:
				wg.Done()
			}
		}
	})()

	wg.Add(1)
	go runner.Start(cmdId)
	wg.Add(1)
	go runner.Start(cmd2id) // Or .StartAll()

	wg.Wait()
}
