# goproc

Process manager library

* start processes with given arguments and environment variables; `.AddCommand` and `.Start` or `.StartAll`
* stop them; `.Signal(cmdId,os.Kill)` or `.Cleanup` to kill all process
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
    OnExit: func(cmd *goproc.Cmd) {
        return
    },
})

daemon.Start(cmdId)
```

## TODO

* comments and documentation in code;
* continuous integration configuration;
* integration tests;
* unit tests.
