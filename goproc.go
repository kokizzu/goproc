package goproc

import (
	"bufio"
	"fmt"
	"github.com/kokizzu/gotro/I"
	"github.com/kokizzu/gotro/L"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)
import "os/exec"
import "log"
import "sync"

type CommandId int

type StringCallback func(*Cmd, string) error

type ParameterlessCallback func(*Cmd)

type IntReturningCallback func(*Cmd) int64

type CmdState int

const (
	NoRestart      = 0
	RestartForever = -1
)

const (
	NotStarted CmdState = iota
	Started
	Killed
	Crashed
	Exited
)

type Cmd struct {
	Program            string
	InheritEnv         bool                 // inherit current console's env
	Env                []string             // environment variables
	Parameters         []string             // program parameters
	StartDelayMs       int64                // delay before starting process
	RestartDelayMs     int64                // delay before restarting process, <0 if you don't want to restart this process
	OnRestart          IntReturningCallback // this overwrites RestartDelayMs
	HideStdout         bool
	OnStdout           StringCallback
	HideStderr         bool
	OnStderr           StringCallback
	OnExit             ParameterlessCallback // when max restart reached, or manually killed
	OnProcessCompleted ParameterlessCallback
	MaxRestart         int
	MaxRuntime         int // maximum command execution time
	restartCount       int
	state              CmdState
	strCache           string
	// channel API
	UseChannelApi            bool // if true, you must handle all 4 channels, or they will have old data after Exit then Start again (eg. if first Start channel not handled, then Exit, and then Start again but with channel handled, the channel would still hold first run's data)
	StdoutChannel            chan string
	StderrChannel            chan string
	ProcesssCompletedChannel chan bool
	ExitChannel              chan bool
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

type Process struct {
	exe *exec.Cmd
}

type Goproc struct {
	cmds  []*Cmd
	procs []*Process
	lock  sync.Mutex
}

func New() *Goproc {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	res := &Goproc{
		cmds: []*Cmd{},
	}
	go func() {
		<-c
		res.Cleanup()
		os.Exit(1)
	}()

	return res
}

// add a new command to run, not yet started until Start called
// returns command id
func (g *Goproc) AddCommand(cmd *Cmd) CommandId {
	g.lock.Lock()
	defer g.lock.Unlock()
	g.cmds = append(g.cmds, cmd)
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

// send signal to process
func (g *Goproc) Signal(cmdId CommandId, signal os.Signal) error {
	idx := int(cmdId)
	if idx >= len(g.cmds) || idx < 0 {
		return fmt.Errorf(`invalid command index, should be zero to %d`, len(g.cmds)-1)
	}

	prefix := `cmd` + I.ToStr(idx) + `: `

	cmd := g.cmds[idx]
	proc := g.procs[idx]

	log.Printf(prefix+`signalling %s\n`, cmd)

	if cmd.state != NotStarted {
		if signal == os.Kill {
			// * stop them; signal=os.Kill
			cmd.state = Killed
			err := proc.exe.Process.Kill()
			if L.IsError(err, `error proc.exe.Process.Kill`) {
				return err
			}
		} else {
			// * relay termination signals;
			err := proc.exe.Process.Signal(signal)
			if L.IsError(err, `error proc.exe.Process.Signal %d`, signal) {
				return err
			}
		}
	}
	return nil
}

// start certain command
func (g *Goproc) Start(cmdId CommandId) error {
	idx := int(cmdId)
	if idx >= len(g.cmds) || idx < 0 {
		return fmt.Errorf(`invalid command index, should be zero to %d`, len(g.cmds)-1)
	}

	prefix := `cmd` + I.ToStr(idx) + `: `

	cmd := g.cmds[idx]

	if cmd.state != NotStarted {
		return fmt.Errorf(`invalid command state=%d already started`, cmd.state)
	}

	time.Sleep(time.Millisecond * time.Duration(cmd.StartDelayMs))

	for {
		// refill process
		proc := g.procs[idx]
		proc.exe = exec.Command(cmd.Program, cmd.Parameters...)
		if cmd.InheritEnv {
			proc.exe.Env = append(os.Environ(), cmd.Env...)
		} else {
			proc.exe.Env = cmd.Env
		}

		// get output buffer and start
		stderr, err := proc.exe.StderrPipe()
		if L.IsError(err, prefix+`error proc.exe.StderrPipe %s`, cmd) {
			return err
		}
		stdout, err := proc.exe.StdoutPipe()
		if L.IsError(err, prefix+`error proc.exe.StdoutPipe %s`, cmd) {
			return err
		}
		log.Printf(prefix + ` starting: ` + cmd.String())
		err = proc.exe.Start()
		if L.IsError(err, prefix+`error proc.exe.Start %s`, cmd) {
			return err
		}
		cmd.state = Started

		// use channel API
		if cmd.UseChannelApi {
			go (func() {
				scanner := bufio.NewScanner(stderr)
				scanner.Split(bufio.ScanLines)
				for scanner.Scan() {
					cmd.StderrChannel <- scanner.Text()
				}
			})()
			go (func() {
				scanner := bufio.NewScanner(stdout)
				scanner.Split(bufio.ScanLines)
				for scanner.Scan() {
					cmd.StdoutChannel <- scanner.Text()
				}
			})()
		}
		// call callback or pipe
		// * read their stdout and stderr;
		hasOutCallback := cmd.OnStdout != nil
		if hasOutCallback || !cmd.HideStdout {
			go (func() {
				scanner := bufio.NewScanner(stderr)
				scanner.Split(bufio.ScanLines)
				for scanner.Scan() {
					line := scanner.Text()
					if hasOutCallback {
						err = cmd.OnStdout(cmd, line)
						L.IsError(err, prefix+`error OnStderr: `+line)
					}
					if !cmd.HideStdout {
						log.Println(prefix + line)
					}
				}
			})()
		}

		hasErrCallback := cmd.OnStderr != nil
		if hasErrCallback || !cmd.HideStdout {
			go (func() {
				scanner := bufio.NewScanner(stdout)
				scanner.Split(bufio.ScanLines)
				for scanner.Scan() {
					line := scanner.Text()
					if hasErrCallback {
						err = cmd.OnStdout(cmd, line)
						L.IsError(err, prefix+`error OnStdout: `+line)
					}
					if !cmd.HideStdout {
						log.Println(prefix + line)
					}
				}
			})()
		}

		// wait for exit
		err = proc.exe.Wait()
		if L.IsError(err, prefix+`error proc.exe.Wait %s`, cmd) {
			if cmd.state != Killed {
				cmd.state = Crashed
			}
		} else {
			log.Println("exited")
			if cmd.state != Killed {
				cmd.state = Exited
			}
		}

		if cmd.OnProcessCompleted != nil {
			cmd.OnProcessCompleted(cmd)
			if cmd.UseChannelApi {
				go (func() {
					cmd.ProcesssCompletedChannel <- true
				})()
			}
		}

		// * restart them when they crash;
		cmd.restartCount += 1
		if cmd.MaxRestart > RestartForever && cmd.restartCount > cmd.MaxRestart {
			log.Printf(prefix+`max restart reached %d\n`, cmd.MaxRestart)
			break
		}

		delayMs := cmd.RestartDelayMs
		if cmd.OnRestart != nil {
			delayMs = cmd.OnRestart(cmd)
		}
		time.Sleep(time.Millisecond * time.Duration(delayMs))

		log.Printf(prefix+`restarting.. x%d %s\n`, cmd.restartCount, cmd)
	}

	cmd.state = NotStarted
	cmd.restartCount = 0

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

// start all that not yet started
func (g *Goproc) StartAll() {
	g.lock.Lock()
	defer g.lock.Unlock()
	for idx, cmd := range g.cmds {
		if cmd.state == NotStarted {
			g.Start(CommandId(idx))
		}
	}
}

// kill all process
func (g *Goproc) Cleanup() {
	g.lock.Lock()
	defer g.lock.Unlock()

	for idx := range g.cmds {
		g.Kill(CommandId(idx))
	}
}
