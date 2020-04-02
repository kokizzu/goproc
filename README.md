# goproc

Process manager library

* start processes with given arguments and environment variables; `.Start`
* stop them; `.Signal(cmdId,os.Kill)`
* restart them when they crash;  
* relay termination signals; `.Signal(cmdId, ...)`
* read their stdout and stderr; `.OnStdout, .OnStdErr`
* should work on Linux and macOS (not tested on mac).
* ability to stop processes when main processes are SIGKILL'ed `.Cleanup` called when main process killed;
* see more example on `example/` for other demonstrates the usage;
* configurable backoff strategy for restarts; `OnRestart` callback

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
