# goproc

Simple process manager helper library, features:

* start processes `.AddCommand(Cmd)` and `.Start(cmdId)` or `.StartAll()`, with environment variables `Cmd.Env=[]string{}` and `Cmd.InheritEnv=true` 
* stop them; `.Kill(cmdId)` or `.Cleanup()` to kill all process
* restart them when they crash; using `Cmd.RestartDelayMs=1000` (=1s, default is 0) and `Cmd.MaxRestart=5` (restart 5x if process ended/crashed)
* relay termination signals; `.Signal(cmdId, os.Kill)`
* read their stdout and stderr; `Cmd.OnStdout`, `Cmd.OnStdErr` callback
* ability to stop processes when main processes are SIGKILL'ed: `.Cleanup()` called automatically when main process killed;
* configurable backoff strategy for restarts; you can use `Cmd.OnRestart` callback to return random delay or implement your own exponential backoff, setting this callback will render `Cmd.RestartDelayMs` unusable
* `Cmd.OnExit` callback when no more restart reached, you can call `.Start(cmdId)` manually again after this
* `Cmd.OnProcessCompleted` callback each time program completed once (before restarting if MaxRestart not yet reached)
* `Cmd.StartDelayMs=1000` (=1s, default is 0) for delaying start, in milliseconds
* `Cmd.UseChannelApi=true`, if enabled, you can receive from channels: `Cmd.StderrChannel`, `Cmd.OnStdoutChannel`, `Cmd.ProcessCompletedChannel`, `Cmd.ExitChannel` 
* `Cmd.LastExecutionError` property to get last process execution error, check this [answer](//stackoverflow.com/questions/10385551/get-exit-code-go) to get the exit code
* `Cmd.OnStateChanged` callback and `Cmd.StateChangedChannel` channel to track process state
* should work on Linux, and probably MacOS and Windows (untested).
* see [example/](//github.com/kokizzu/goproc/blob/master/example/main.go) for other usage example/demo;

## Versioning

versioning using this format 1.`(M+(YEAR-2021)*12)DD`.`HMM`,
so for example v1.213.1549 means it was released at `2021-02-13 15:49`

## Example

```go

daemon := goproc.New()

cmdId := daemon.AddCommand(&goproc.Cmd{
    Program: `sleep`, // program to run
	WorkDir: `/tmp`, // directory to run the program (optional)
    Parameters: []string{`2`}, // command line arguments
    MaxRestart: goproc.RestartForever, // default: NoRestart=0
    OnStderr: func(cmd *goproc.Cmd, line string) error { // optional
        fmt.Println(`OnStderr: `+line)
        return nil
    },
    OnStdout: func(cmd *goproc.Cmd, line string) error { // optional
        fmt.Println(`OnStdout: `+line)
        return nil
    },
})

daemon.Start(cmdId) // use "go" keyword if you need non-blocking version
// ^ by default will only run next command if previous command exited
// if you want to run them in parralel, use daemon.StartParallel().Wait()

daemon.CommandString(cmdId) // returns "sleep 2"
```

## FAQ

Q: Why not just channel? why callback?

A: Because channel requires a consumer, or it would stuck/block if channel is full, while callback doesn't (this is why I add flag to activate the channel API). To the Percona reviewer that rejected me because I didn't use channel at the first time "the whole thing is written in JavaScript translated to Go, not in Go. Technical task does not adhere to Go best practices which is what would expect from the candidate. One example was that the code is written in JS-like style (e.g. callback). Go code does not use callback like node.js for example", well, jokes on you XD

Q: How to ignore error being printed?

A: assign `Goproc.HasErrFunc` with `goproc.DiscardHasErr`, other option are: `L.IsError` (default), `goproc.LogHasErr` (uses log), `goproc.PrintHasErr` (uses fmt), or you can always create your own.

## TODO

* implement `.Pause` and `.Resume` API
* comments and documentation in code;
* continuous integration configuration;
* integration tests;
* unit tests.
