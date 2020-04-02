# goproc

Simple process manager library, features:

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
* `Cmd.UseChannelApi=true`, if enabled, you must handle all 4 channels: `Cmd.StderrChannel`, `Cmd.OnStdoutChannel`, `Cmd.ProcessCompletedChannel`, `Cmd.ExitChannel` 
* should work on Linux and MacOS (untested tho).
* see [example/](//github.com/kokizzu/goproc/blob/master/example/main.go) for other usage example/demo;

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
