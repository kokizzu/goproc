# goproc

Process manager library. Features:

* start processes with given arguments and environment variables; `.AddCommand` and `.Start` or `.StartAll`
* stop them; `.Kill(cmdId)` or `.Cleanup` to kill all process
* restart them when they crash; `.RestartDelayMs` and `.MaxRestart`
* relay termination signals; `.Signal(cmdId, ...)`
* read their stdout and stderr; `.OnStdout, .OnStdErr`
* should work on Linux and macOS (not tested on mac).
* ability to stop processes when main processes are SIGKILL'ed `.Cleanup` called automatically when main process killed;
* see more example on `example/` for other demonstrating the usage;
* configurable backoff strategy for restarts; you can use `OnRestart` callback to return random delay or implement exponential backoff

## Example

```

daemon := goproc.New()

cmdId := daemon.AddCommand(&goproc.Cmd{
    Program: `sleep`,
    Parameters: []string{`2`},
    MaxRestart: goproc.RestartForever,
    OnStderr: func(cmd *goproc.Cmd, s string) error {
        fmt.Println(`OnStderr: `+s)
        return nil
    },
    OnStdout: func(cmd *goproc.Cmd, s string) error {
        fmt.Println(`OnStdout: `+s)
        return nil
    },
})

daemon.Start(cmdId) // use go if you need non-blocking version

```

## TODO

* implement .Pause and .Resume API
* implement `.OnStderr(cmdId)`, `.OnStdout(cmdId)`, `.OnProcessCompleted(cmdId)`, `.OnExit(cmdId)` Channel API (use MultiReader and 2 more goroutine (one for stdin, one for stderr) so it won't block current callback and logger)
* comments and documentation in code;
* continuous integration configuration;
* integration tests;
* unit tests.
