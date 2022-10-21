package goproc

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/kokizzu/gotro/A"
	"github.com/kokizzu/gotro/S"

	"github.com/kokizzu/gotro/I"
	"github.com/kokizzu/gotro/L"
)
import "os/exec"
import "log"
import "sync"

type CommandId int
type CmdState int

type StringCallback func(*Cmd, string) error
type IntCallback func(*Cmd, int64)
type CmdStateCallback func(*Cmd, CmdState, CmdState)
type ParameterlessCallback func(*Cmd)

type IntReturningCallback func(*Cmd) int64

const (
	NoRestart      = 0
	RestartForever = -1
)

const (
	NotStarted CmdState = iota // program can be started (again)
	Started                    // program running
	Killed                     // program killed using API
	Crashed                    // program terminated with error
	Exited                     // program terminated without error
)

type Cmd struct {
	Program    string   // program name, could be full path or only the program name, depends on PATH environment variables
	Parameters []string // program parameters
	WorkDir    string   // starting directory

	PrefixLabel string // prefix label instead of goprocID

	InheritEnv bool     // inherit current console's env
	Env        []string // environment variables

	StartDelayMs   int64 // delay before starting process
	RestartDelayMs int64 // delay before restarting process, <0 if you don't want to restart this process

	HideStdout bool // disable stdout logging
	HideStderr bool // disable stderr logging

	MaxRestart         int   // -1 = always restart, 0 = only run once, >0 run N times
	LastExecutionError error // last execution error, useful for OnProcessCompleted or ProcessCompletedChannel
	RestartCount       int   // can be overwritten for early exit or restart from 0

	OnStdout           StringCallback        // one line fetched from stdout
	OnStderr           StringCallback        // one line fetched from stderr
	OnRestart          IntReturningCallback  // this overwrites RestartDelayMs
	OnExit             ParameterlessCallback // when max restart reached, or manually killed
	OnProcessCompleted IntCallback           // when 1x process done, can be restarting depends on RestartCount and MaxCount
	OnStateChanged     CmdStateCallback      // triggered when stated changed

	state    CmdState
	strCache string

	// channel API
	UseChannelApi            bool
	StdoutChanLength         int
	StdoutChannel            chan string
	StderrChanLength         int
	StderrChannel            chan string
	ProcesssCompletedChannel chan int64
	ExitChannel              chan bool
	StateChangedChannel      chan CmdState
}

func (cmd *Cmd) String() string {
	if len(cmd.Parameters) == 0 {
		return cmd.Program
	}
	if len(cmd.strCache) > 0 {
		return cmd.strCache
	}

	// escape parameters
	cmd.strCache += cmd.Program
	arr := []string{}
	for _, param := range cmd.Parameters {
		if strings.Contains(param, `"`) {
			param = strings.Replace(param, `"`, `\"`, -1)
		}
		arr = append(arr, param)
	}
	cmd.strCache += ` "` + strings.Join(arr, `" "`) + `"`
	return cmd.strCache
}

func (g *Cmd) GetState() CmdState {
	return g.state
}

func (cmd *Cmd) setState(newState CmdState) {
	oldState := cmd.state
	cmd.state = newState
	if cmd.OnStateChanged != nil {
		cmd.OnStateChanged(cmd, oldState, newState)
	}
	if cmd.UseChannelApi {
		go (func() {
			cmd.StateChangedChannel <- newState
		})()
	}
}

type Process struct {
	exe *exec.Cmd
}

type Goproc struct {
	cmds       []*Cmd
	procs      []*Process
	lock       sync.Mutex
	HasErrFunc func(err error, fmt string, args ...any) bool
}

// LogHasErr to log if error occurred, must return true if err not nil
func LogHasErr(err error, fmt string, args ...any) bool {
	if err != nil {
		log.Printf(fmt, args...)
		return true
	}
	return false
}

// PrintHasErr to log using fmt if error occurred, must return true if err not nil
func PrintHasErr(err error, msg string, args ...any) bool {
	if err != nil {
		fmt.Printf(msg+"\n", args...)
		return true
	}
	return false
}

// DiscardHasErr to ignore if error occurred, must return true if err not nil
func DiscardHasErr(err error, _ string, _ ...any) bool {
	return err != nil
}

func New() *Goproc {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	res := &Goproc{
		cmds:       []*Cmd{},
		HasErrFunc: L.IsError,
	}

	go func() {
		<-c
		res.Cleanup()
		os.Exit(1)
	}()

	return res
}

// AddCommand add a new command to run, not yet started until Start called
// returns command id
func (g *Goproc) AddCommand(cmd *Cmd) CommandId {
	g.lock.Lock()
	defer g.lock.Unlock()
	g.cmds = append(g.cmds, cmd)
	cmd.StdoutChannel = make(chan string, cmd.StdoutChanLength)
	cmd.StderrChannel = make(chan string, cmd.StderrChanLength)
	cmd.ExitChannel = make(chan bool)
	cmd.ProcesssCompletedChannel = make(chan int64)
	cmd.StateChangedChannel = make(chan CmdState)
	cmd.state = NotStarted
	// * start processes with given arguments and environment variables;
	g.procs = append(g.procs, &Process{
		exe: nil,
	})
	cmdId := len(g.cmds) - 1
	return CommandId(cmdId)
}

func (g *Goproc) Kill(cmdId CommandId) error {
	return g.Signal(cmdId, os.Kill)
}

// Signal send signal to process
func (g *Goproc) Signal(cmdId CommandId, signal os.Signal) error {
	idx := int(cmdId)
	if idx >= len(g.cmds) || idx < 0 {
		return fmt.Errorf(`invalid command index, should be zero to %d`, len(g.cmds)-1)
	}

	prefix := `cmd` + I.ToStr(idx) + `: `

	cmd := g.cmds[idx]
	proc := g.procs[idx]

	log.Printf(prefix+`signalling %s\n`, cmd)

	if cmd.state != Started {
		return fmt.Errorf(`process not started: %d`, cmd.state)
	}
	if signal == os.Kill {
		// * stop them; signal=os.Kill
		err := proc.exe.Process.Kill()
		cmd.setState(Killed)
		if g.HasErrFunc(err, `error proc.exe.Process.Kill`) {
			return err
		}
	} else {
		// * relay termination signals;
		err := proc.exe.Process.Signal(signal)
		if g.HasErrFunc(err, `error proc.exe.Process.Signal %d`, signal) {
			return err
		}
	}
	return nil
}

// Start start certain command
func (g *Goproc) Start(cmdId CommandId) error {
	idx := int(cmdId)
	if idx >= len(g.cmds) || idx < 0 {
		return fmt.Errorf(`invalid command index, should be zero to %d`, len(g.cmds)-1)
	}

	cmd := g.cmds[idx]
	cmd.strCache = `` // reset cache

	prefix := S.IfEmpty(cmd.PrefixLabel, `CMD:`+I.ToStr(idx)) + `: `

	if cmd.state != NotStarted {
		return fmt.Errorf(`invalid command state=%d already started`, cmd.state)
	}

	time.Sleep(time.Millisecond * time.Duration(cmd.StartDelayMs))

	for {
		// refill process
		proc := g.procs[idx]
		proc.exe = exec.Command(cmd.Program, cmd.Parameters...)
		proc.exe.Dir = cmd.WorkDir
		if cmd.InheritEnv {
			proc.exe.Env = append(os.Environ(), cmd.Env...)
		} else {
			proc.exe.Env = cmd.Env
		}

		// get output buffer and start
		stderr, err := proc.exe.StderrPipe()
		if g.HasErrFunc(err, prefix+`error proc.exe.StderrPipe %s`, cmd) {
			return err
		}
		stdout, err := proc.exe.StdoutPipe()
		if g.HasErrFunc(err, prefix+`error proc.exe.StdoutPipe %s`, cmd) {
			return err
		}
		log.Printf(prefix + `starting: ` + cmd.String())
		start := time.Now()
		err = proc.exe.Start()
		if g.HasErrFunc(err, prefix+`error proc.exe.Start %s`, cmd) {
			return err
		}
		cmd.setState(Started)

		if cmd.UseChannelApi {
			go (func() {
				scanner := bufio.NewScanner(stdout)
				scanner.Split(bufio.ScanLines)
				for scanner.Scan() {
					cmd.StdoutChannel <- scanner.Text()
				}
			})()
			go (func() {
				scanner := bufio.NewScanner(stderr)
				scanner.Split(bufio.ScanLines)
				for scanner.Scan() {
					cmd.StderrChannel <- scanner.Text()
				}
			})()
		}

		// call callback or pipe
		// * read their stdout and stderr;
		hasErrCallback := cmd.OnStderr != nil
		if hasErrCallback || !cmd.HideStdout || cmd.UseChannelApi {
			go (func() {
				scanner := bufio.NewScanner(stderr)
				scanner.Split(bufio.ScanLines)
				for scanner.Scan() {
					line := scanner.Text()
					if hasErrCallback {
						err = cmd.OnStdout(cmd, line)
						g.HasErrFunc(err, prefix+`error OnStderr: `+line)
					}
					if !cmd.HideStdout {
						log.Println(prefix + line)
					}
				}
			})()
		}

		hasOutCallback := cmd.OnStdout != nil
		if hasOutCallback || !cmd.HideStdout || cmd.UseChannelApi {
			go (func() {
				scanner := bufio.NewScanner(stdout)
				scanner.Split(bufio.ScanLines)
				for scanner.Scan() {
					line := scanner.Text()
					if hasOutCallback {
						err = cmd.OnStdout(cmd, line)
						g.HasErrFunc(err, prefix+`error OnStdout: `+line)
					}
					if !cmd.HideStdout {
						log.Println(prefix + line)
					}
				}
			})()
		}

		// wait for exit
		err = proc.exe.Wait()
		if g.HasErrFunc(err, prefix+`error proc.exe.Wait %s`, cmd) {
			if cmd.state != Killed {
				cmd.setState(Crashed)
			}
		} else {
			log.Println("exited")
			if cmd.state != Killed {
				cmd.setState(Exited)
			}
		}
		cmd.LastExecutionError = err

		if cmd.OnProcessCompleted != nil {
			durationMs := time.Since(start).Milliseconds()
			cmd.OnProcessCompleted(cmd, durationMs)
			if cmd.UseChannelApi {
				go (func() {
					cmd.ProcesssCompletedChannel <- durationMs
				})()
			}
		}

		// * restart them when they crash;
		cmd.RestartCount += 1
		if cmd.MaxRestart > RestartForever && cmd.RestartCount > cmd.MaxRestart {
			log.Printf(prefix+`max restart reached %d`, cmd.MaxRestart)
			break
		}

		delayMs := cmd.RestartDelayMs
		if cmd.OnRestart != nil {
			delayMs = cmd.OnRestart(cmd)
		}
		time.Sleep(time.Millisecond * time.Duration(delayMs))

		log.Printf(prefix+`restarting.. x%d %s`, cmd.RestartCount, cmd)
	}

	cmd.setState(NotStarted)
	cmd.RestartCount = 0

	if cmd.OnExit != nil {
		cmd.OnExit(cmd)
	}
	if cmd.UseChannelApi {
		go (func() {
			cmd.ExitChannel <- true
		})()
	}

	return nil
}

// StartAll start all that not yet started
func (g *Goproc) StartAll() {
	g.lock.Lock()
	defer g.lock.Unlock()
	for idx, cmd := range g.cmds {
		if cmd.state == NotStarted {
			g.Start(CommandId(idx))
		}
	}
}

// StartAllParallel start all that not yet started in parallel
func (g *Goproc) StartAllParallel() *sync.WaitGroup {
	g.lock.Lock()
	defer g.lock.Unlock()
	wg := &sync.WaitGroup{}
	for idx, cmd := range g.cmds {
		if cmd.state == NotStarted {
			wg.Add(1)
			if cmd.OnExit != nil {
				onExitCopy := cmd.OnExit
				cmd.OnExit = func(cmd *Cmd) {
					onExitCopy(cmd)
					wg.Done()
				}
			} else {
				cmd.OnExit = func(cmd *Cmd) {
					wg.Done()
				}
			}
			go g.Start(CommandId(idx))
		}
	}
	return wg
}

// Cleanup kill all process
func (g *Goproc) Cleanup() {
	g.lock.Lock()
	defer g.lock.Unlock()

	for idx := range g.cmds {
		g.HasErrFunc(g.Kill(CommandId(idx)), "")
	}
}

// Terminate kill program
func (g *Goproc) Terminate(cmdId CommandId) error {
	return exec.Command(`kill`, I.ToS(int64(g.procs[cmdId].exe.Process.Pid))).Run()
}

// CommandString return the command string with agruments
func (g *Goproc) CommandString(cmdId CommandId) string {
	if cmdId < 0 || cmdId >= CommandId(len(g.cmds)) {
		return ``
	}
	cmd := g.cmds[cmdId]
	return cmd.Program + ` ` + A.StrJoin(cmd.Parameters, ` `)
}
