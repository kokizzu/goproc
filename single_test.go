package goproc

import (
	"fmt"
	"testing"

	"github.com/zeebo/assert"
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

func TestRun1(t *testing.T) {
	stdout, stderr, err, exitCode := Run1(&Cmd{
		Program:    `echo`,
		Parameters: []string{`1`},
	})
	assert.Equal(t, "1\n", stdout)
	assert.Equal(t, "", stderr)
	assert.NoError(t, err)
	assert.Equal(t, 0, exitCode)
}

func TestRun1missing(t *testing.T) {
	stdout, stderr, err, exitCode := Run1(&Cmd{
		Program:    `not_found`,
		Parameters: []string{`1`},
	})
	assert.Equal(t, "", stdout)
	assert.Equal(t, "", stderr)
	assert.Error(t, err)
	assert.Equal(t, 0, exitCode)
}

func TestRun1long(t *testing.T) {
	_, stderr, err, exitCode := Run1(&Cmd{
		Program:    `ls`,
		Parameters: []string{`-alr`},
	})
	assert.Equal(t, "", stderr)
	assert.NoError(t, err)
	assert.Equal(t, 0, exitCode)
}
