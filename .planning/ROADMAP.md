# Roadmap: AP CLI -- Machine-Friendly Output

## Overview

This milestone adds structured JSON output to the AP CLI so that AI agents and scripts can programmatically interact with the Agent Platform. The work splits into two phases: first building the shared output infrastructure (helpers, error handling, compatibility guardrails), then wiring `--json` into every target command. The foundation phase is small but critical -- every command depends on its output helpers and error wrapper.

## Phases

**Phase Numbering:**
- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

- [x] **Phase 1: Output Infrastructure** - Shared JSON output helpers, error handling, and compatibility guardrails
- [ ] **Phase 2: Command JSON Output** - Wire `--json` flag into all target commands

## Phase Details

### Phase 1: Output Infrastructure
**Goal**: A reusable output layer exists so that any command can produce structured JSON with one function call, and errors are automatically intercepted and formatted
**Depends on**: Nothing (first phase)
**Requirements**: OUT-01, OUT-02, OUT-03, OUT-04, OUT-05, ERR-01, ERR-02, COMPAT-01, COMPAT-02
**Success Criteria** (what must be TRUE):
  1. A command can register a `--json` flag and call a single helper to write its result as valid JSON to stdout
  2. When `--json` is active and a command errors, structured JSON error output appears on stderr with a non-zero exit code
  3. List results serialize as `[]` (not `null`) when the collection is empty
  4. Running any command without `--json` produces identical output to the current CLI (no regressions)
  5. The `actions do --json` flag collision is resolved so that adding output `--json` to actions commands will not conflict
**Plans**: 2 plans

Plans:
- [x] 01-01-PLAN.md -- Create internal/output package (JSON helpers + error formatting)
- [x] 01-02-PLAN.md -- Wire --json flag, Execute() error interceptor, --body rename

### Phase 2: Command JSON Output
**Goal**: Every target command produces correct, complete structured JSON when invoked with `--json`, making the CLI fully usable by AI agents and automation scripts
**Depends on**: Phase 1
**Requirements**: CMD-01, CMD-02, CMD-03, CMD-04, CMD-05, CMD-06, CMD-07, CMD-08, CMD-09
**Success Criteria** (what must be TRUE):
  1. `ap agents list --json` and `ap agents get --json` output valid JSON (array and object respectively) that an external tool can parse without error
  2. `ap sessions list --json`, `ap org --json`, `ap workspace list --json`, `ap integrations list --json`, `ap permissions --json`, and `ap actions list --json` each output valid JSON matching their data shape
  3. `ap auth status --json` outputs the current auth state as a JSON object (login/logout flows remain unchanged)
  4. No human-readable text leaks to stdout for any command when `--json` is active
**Plans**: TBD

Plans:
- [ ] 02-01: TBD
- [ ] 02-02: TBD

## Progress

**Execution Order:**
Phases execute in numeric order: 1 -> 2

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Output Infrastructure | 2/2 | Complete | 2026-03-09 |
| 2. Command JSON Output | 0/0 | Not started | - |
