package goproc

import (
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/zeebo/assert"
)

func TestParallel(t *testing.T) {
	proc := NewWithCleanup()
	proc.AddCommand(&Cmd{
		Program:    `echo`,
		Parameters: []string{`1`},
		MaxRestart: 4,
	})
	proc.AddCommand(&Cmd{
		Program:    `echo`,
		Parameters: []string{`2`},
		MaxRestart: 4,
	})
	proc.StartAllParallel().Wait()
}

func TestParallelMultiple(t *testing.T) {
	proc := NewWithCleanup()
	var total uint32
	firstCmdId := proc.AddCommand(&Cmd{
		Program:    `echo`,
		Parameters: []string{`-1`},
		MaxRestart: 4,
		OnProcessCompleted: func(cmd *Cmd, durationMs int64) {
			atomic.AddUint32(&total, 1)
			time.Sleep(100 * time.Millisecond)
		},
	})
	wg := proc.StartAllParallel()
	const n = 10
	for z := range n {
		go func() {
			cmdId := proc.AddCommand(&Cmd{
				Program:    `echo`,
				Parameters: []string{strconv.Itoa(z)},
				MaxRestart: 4,
				OnProcessCompleted: func(cmd *Cmd, durationMs int64) {
					atomic.AddUint32(&total, 1)
				},
			})
			proc.Start(cmdId)
		}()
	}
	time.Sleep(500 * time.Millisecond)
	wg.Wait()
	assert.Equal(t, uint32(5+n*5), total)

	// call again once
	proc.Start(firstCmdId)
	assert.Equal(t, uint32(5+n*5+5), total)

}
