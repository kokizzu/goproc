package main

import (
	"fmt"
	"github.com/kokizzu/goproc"
	"github.com/kokizzu/gotro/I"
	"math/rand"
	"sync"
)

func main() {
	wg := sync.WaitGroup{}

	runner := goproc.New()

	// callback API demo
	cmdId := runner.AddCommand(&goproc.Cmd{
		Program:      `echo`,
		Parameters:   []string{`123`},
		StartDelayMs: 1000,
		MaxRestart:   goproc.RestartForever,
		OnStderr: func(cmd *goproc.Cmd, s string) error {
			fmt.Println(`OnStderr0: ` + s)
			return nil
		},
		OnStdout: func(cmd *goproc.Cmd, s string) error {
			fmt.Println(`OnStdout0: ` + s)
			return nil
		},
		OnProcessCompleted: func(cmd *goproc.Cmd, durationMs int64) {
			fmt.Println(`OnProcessCompleted0, done in ` + I.ToS(durationMs) + `ms`)
		},
		OnRestart: func(cmd *goproc.Cmd) int64 {
			sleep := 1000 + rand.Int63()%3000
			fmt.Println(`OnRestart, sleep for ` + I.ToS(sleep) + `ms`)
			return sleep
		},
		OnExit: func(cmd *goproc.Cmd) {
			fmt.Println(`OnExit0`)
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
			case durationMs := <-cmd.ProcesssCompletedChannel:
				fmt.Println(`ProcesssCompletedChannel1, done in ` + I.ToS(durationMs) + `ms`)
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
