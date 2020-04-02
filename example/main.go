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
		Program:        `echo`,
		Parameters:     []string{`123`},
		RestartDelayMs: 2000,
		MaxRestart:     goproc.RestartForever,
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
		UseChannelApi:  true,
		HideStdout:     true,
		HideStderr:     true,
		// callback API also still can be used
	}
	cmd2id := runner.AddCommand(cmd)

	go (func() {
		for {
			select {
			case line := <-cmd.StdoutChannel:
				fmt.Println(`StdoutChannel1: ` + line)
			case line := <-cmd.StderrChannel:
				fmt.Println(`StderrChannel1: ` + line)
			case <-cmd.ProcesssCompletedChannel:
				fmt.Println(`ProcesssCompletedChannel1`)
			case <-cmd.ExitChannel:
				fmt.Println(`ExitChannel1`)
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
