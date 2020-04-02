# goproc

Process manager library. Features:

* start processes `.AddCommand` and `.Start` or `.StartAll`, with environment variables `Cmd.Env` and `Cmd.InheritEnv` 
* stop them; `.Kill(cmdId)` or `.Cleanup` to kill all process
* restart them when they crash; using `Cmd.RestartDelayMs` and `Cmd.MaxRestart` property
* relay termination signals; `.Signal(cmdId, ANYSIGNAL)`
* read their stdout and stderr; `Cmd.OnStdout`, `Cmd.OnStdErr`
* should work on Linux and macOS (untested on macOS tho).
* ability to stop processes when main processes are SIGKILL'ed `.Cleanup` called automatically when main process killed;
* see more example on `example/` for other usage demo;
* configurable backoff strategy for restarts; you can use `Cmd.OnRestart` callback to return random delay or implement your own exponential backoff
* `Cmd.OnExit` when no more restart reached
* `Cmd.OnProcessCompleted` callback each time program completed once (before restarting if MaxRestart not yet reached)
* `Cmd.StartDelayMs` for delaying start
* `Cmd.UseChannelApi`, if enabled, you must handle all 4 channel: `Cmd.StderrChannel`, `Cmd.OnStdoutChannel`, `Cmd.ProcessCompletedchannel`, `Cmd.ExitChannel` 

## Example

```

daemon := goproc.New()

cmdId := daemon.AddCommand(&goproc.Cmd{
    Program: `sleep`, // program to run
    Parameters: []string{`2`}, // command line arguments
    MaxRestart: goproc.RestartForever, // default: NoRestart=0
    OnStderr: func(cmd *goproc.Cmd, s string) error { // optional
        fmt.Println(`OnStderr: `+s)
        return nil
    },
    OnStdout: func(cmd *goproc.Cmd, s string) error { // optional
        fmt.Println(`OnStdout: `+s)
        return nil
    },
})

daemon.Start(cmdId) // use go if you need non-blocking version

```

## TODO

* implement `.Pause` and `.Resume` API
* comments and documentation in code;
* continuous integration configuration;
* integration tests;
* unit tests.
