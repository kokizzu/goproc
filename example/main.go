package main

import (
	"fmt"
	"github.com/kokizzu/goproc"
	"sync"
)

func main() {
	wg := sync.WaitGroup{}

	runner := goproc.New()

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
	cmd2id := runner.AddCommand(&goproc.Cmd{
		Program:        `ps`,
		Parameters:     []string{`ux`},
		RestartDelayMs: 3000,
		MaxRestart:     2,
		OnStderr: func(cmd *goproc.Cmd, s string) error {
			fmt.Println(`OnStderr1: ` + s)
			return nil
		},
		OnStdout: func(cmd *goproc.Cmd, s string) error {
			fmt.Println(`OnStdout1: ` + s)
			return nil
		},
		OnExit: func(cmd *goproc.Cmd) {
			wg.Done()
			return
		},
	})

	wg.Add(1)
	go runner.Start(cmdId)
	wg.Add(1)
	go runner.Start(cmd2id)

	wg.Wait()
}
