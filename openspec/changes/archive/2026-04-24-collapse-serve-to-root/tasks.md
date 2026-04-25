## 1. Merge serve into root

- [x] 1.1 Move `RunE` and flag registration from `cmd/serve.go` into `cmd/root.go` (set `rootCmd.RunE` and call `config.BindFlags(rootCmd.Flags())` in `init`)
- [x] 1.2 Delete `cmd/serve.go`
- [x] 1.3 Set `rootCmd.Args = cobra.NoArgs` so stray positional args are rejected

## 2. Verify

- [x] 2.1 `go build ./...` and `go test ./...` pass
- [x] 2.2 `./proxy --upstream-host=api.example.com --listen-addr=:18080` starts the server and handles a request
- [x] 2.3 `./proxy --help` prints flags; no `serve` listed
