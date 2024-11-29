# slogutils

## Example

```go
// setup global instance
slogutils.SetupGlobalLogger("debug", os.Stdout)
// use as always
slog.Debug("running with config")
// helper func
slogutils.Fatal("could not connect to postgres", "error", err)
```
