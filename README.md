# Agent Platform CLI

`ap` is the terminal client for Agent Platform. It handles WorkOS sign-in, personal and team workspace context, the session TUI, and the control-plane surfaces for integrations, permissions, and gateway actions. User-facing docs live at [docs.agentplatform.cloud](https://docs.agentplatform.cloud).

## Developer Notes

- `cmd/` is the Cobra command tree.
- `internal/tui/` contains the Bubble Tea flows for spawn, resume, and API key setup.
- `internal/controlplane/` is the typed client for the Go control plane API.
- Local state lives in the OS config directory for `ap`:
  macOS `~/Library/Application Support/ap/`
  Linux `~/.config/ap/`
- Production auth and control-plane defaults are built into the CLI. Use environment variables to point a build at non-production services.
- Build with `make build`, install with `make install`, and run tests with `go test ./...`.
