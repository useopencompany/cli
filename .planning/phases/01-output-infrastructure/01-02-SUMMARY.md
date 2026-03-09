---
phase: 01-output-infrastructure
plan: 02
subsystem: output
tags: [json, cobra, flags, error-handling]

# Dependency graph
requires:
  - "01-01: internal/output package with JSON(), JSONTo(), ErrorJSON() helpers"
provides:
  - "--json persistent flag on rootCmd inherited by all subcommands"
  - "Execute() error interceptor: JSON errors to stderr when --json active"
  - "--body flag on doCmd (renamed from --json to resolve collision)"
  - "isJSONOutput() helper for commands to check JSON mode"
affects: [02-command-flags]

# Tech tracking
tech-stack:
  added: []
  patterns: [persistent-flag-pattern, error-interceptor, silence-cobra-output]

key-files:
  created: []
  modified:
    - cmd/root.go
    - cmd/actions.go
    - cmd/root_test.go

key-decisions:
  - "Set SilenceErrors and SilenceUsage globally on rootCmd -- Cobra default 'Error:' prefix is redundant since Execute() handles error display"
  - "isJSONOutput() checks flag.Changed not flag.Value -- ensures default false does not trigger JSON mode"
  - "Added explicit fmt.Fprintf error output in non-JSON path since SilenceErrors suppresses Cobra's default"

patterns-established:
  - "Persistent flag pattern: --json is a persistent bool on rootCmd, checked via isJSONOutput()"
  - "Error interceptor: Execute() wraps rootCmd.Execute() and routes errors to JSON or text based on flag"
  - "Flag collision resolution: local flags that collide with persistent flags get renamed"

requirements-completed: [OUT-01, OUT-05, ERR-02, COMPAT-01, COMPAT-02]

# Metrics
duration: 2min
completed: 2026-03-09
---

# Phase 1 Plan 2: Command Flags and Error Interceptor Summary

**Persistent --json flag with Execute() error interceptor routing errors to JSON stderr, and --body rename resolving doCmd flag collision**

## Performance

- **Duration:** 2 min
- **Started:** 2026-03-09T13:45:56Z
- **Completed:** 2026-03-09T13:47:39Z
- **Tasks:** 2
- **Files modified:** 3

## Accomplishments
- Persistent --json boolean flag on rootCmd inherited by all subcommands for structured output
- Execute() error interceptor writes JSON errors to stderr when --json active, preserves existing behavior otherwise
- Renamed doCmd --json to --body to eliminate flag collision with the new persistent --json
- SilenceErrors/SilenceUsage prevent Cobra text from mixing with JSON output
- 6 new tests covering flag registration, default behavior, silence flags, and body rename

## Task Commits

Each task was committed atomically:

1. **Task 1: Register --json persistent flag and implement Execute() error interceptor** - `bb263b5` (test: RED), `9409eb3` (feat: GREEN)
2. **Task 2: Rename --json to --body on doCmd** - `466e270` (test: RED), `dbb22e9` (feat: GREEN)

_TDD tasks have RED/GREEN commits._

## Files Created/Modified
- `cmd/root.go` - Added --json persistent flag, isJSONOutput(), Execute() error interceptor, SilenceErrors/SilenceUsage
- `cmd/actions.go` - Renamed --json flag to --body on doCmd, updated error message
- `cmd/root_test.go` - 6 new tests for flag registration, defaults, silence flags, body rename

## Decisions Made
- Set SilenceErrors and SilenceUsage globally on rootCmd -- prevents Cobra's "Error:" prefix from mixing with structured JSON output. Safe because Execute() now handles error display for both paths.
- isJSONOutput() checks flag.Changed rather than flag value -- ensures the default `false` value does not accidentally trigger JSON mode when flag is not explicitly passed.
- Added explicit `fmt.Fprintf(os.Stderr, "Error: %s\n", err)` for non-JSON error path since SilenceErrors suppresses Cobra's default error printing.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Full output infrastructure now exists: JSON helpers (Plan 01) + --json flag with error interceptor (Plan 02)
- Any command in Phase 2 can check isJSONOutput() and call output.JSON() / output.ErrorJSON()
- doCmd uses --body for input payload, --json unambiguously means output format

---
*Phase: 01-output-infrastructure*
*Completed: 2026-03-09*
