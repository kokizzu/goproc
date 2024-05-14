package goproc

import (
	"fmt"
	"testing"
)

func TestNotExists(t *testing.T) {
	proc := New()
	proc.AddCommand(&Cmd{
		Program: `not_exists`,
	})
	wg := proc.StartAllParallel()
	wg.Wait()
	fmt.Println(`done`)
}
