package goproc

import (
	"testing"
)

func TestParallel(t *testing.T) {
	proc := New()
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
