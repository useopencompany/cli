---
phase: 01-output-infrastructure
plan: 01
subsystem: output
tags: [json, go, reflect, errors]

# Dependency graph
requires: []
provides:
  - "internal/output package with JSON(), JSONTo(), ErrorJSON() helpers"
  - "Nil/empty slice normalization to [] via reflect"
  - "Structured error JSON with controlplane.APIError extraction"
affects: [02-command-flags]

# Tech tracking
tech-stack:
  added: []
  patterns: [compact-json-output, nil-slice-normalization, error-as-pattern]

key-files:
  created:
    - internal/output/output.go
    - internal/output/output_test.go
    - internal/output/errors.go
    - internal/output/errors_test.go
  modified: []

key-decisions:
  - "Used reflect to detect nil slices and replace with empty slices for [] output"
  - "ErrorJSON uses errors.As to unwrap controlplane.APIError from wrapped errors"

patterns-established:
  - "JSON output pattern: all JSON is compact single-line with trailing newline"
  - "Nil slice normalization: ensureNonNil converts nil slices to []any{} before marshaling"
  - "Error formatting: ErrorJSON extracts APIError fields via errors.As, omits empty code/zero status"

requirements-completed: [OUT-02, OUT-03, OUT-04, ERR-01]

# Metrics
duration: 2min
completed: 2026-03-09
---

# Phase 1 Plan 1: Output Helpers Summary

**Compact JSON output helpers with nil-slice normalization and structured error formatting via errors.As APIError extraction**

## Performance

- **Duration:** 2 min
- **Started:** 2026-03-09T13:41:36Z
- **Completed:** 2026-03-09T13:43:29Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments
- JSON() and JSONTo() write compact JSON with trailing newline, normalizing nil slices to []
- ErrorJSON() formats errors as structured JSON, extracting code/status from controlplane.APIError
- 13 tests covering all behaviors including edge cases (nil slices, empty code, wrapped errors)

## Task Commits

Each task was committed atomically:

1. **Task 1: Create output helpers with tests** - `4eb62b1` (test: RED), `a3bff9c` (feat: GREEN)
2. **Task 2: Create error JSON formatting with tests** - `78f496c` (test: RED), `1b0ec2a` (feat: GREEN)

_TDD tasks have RED/GREEN commits._

## Files Created/Modified
- `internal/output/output.go` - JSON(), JSONTo(), ensureNonNil() helpers
- `internal/output/output_test.go` - 8 tests for JSON output
- `internal/output/errors.go` - ErrorJSON() with APIError extraction
- `internal/output/errors_test.go` - 5 tests for error formatting

## Decisions Made
- Used reflect to detect nil slices at runtime and replace with empty slices -- ensures JSON serialization produces `[]` not `null`
- ErrorJSON uses errors.As for unwrapping, so wrapped APIErrors are still properly extracted

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Output package ready for use by any command in Phase 2
- JSON() for stdout output, JSONTo() for custom writers, ErrorJSON() for stderr

---
*Phase: 01-output-infrastructure*
*Completed: 2026-03-09*
