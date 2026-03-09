# Requirements: AP CLI — Machine-Friendly Output

**Defined:** 2026-03-09
**Core Value:** Any AI agent or script can programmatically interact with the Agent Platform by parsing structured JSON output from CLI commands

## v1 Requirements

### Output Infrastructure

- [ ] **OUT-01**: All non-auth commands accept a `--json` boolean flag
- [x] **OUT-02**: When `--json` is passed, command outputs valid JSON to stdout
- [x] **OUT-03**: List commands output JSON arrays; single-item commands output JSON objects
- [x] **OUT-04**: Empty collections output `[]` (not `null`)
- [ ] **OUT-05**: No human-readable text (fmt.Printf) leaks to stdout when `--json` is active

### Error Handling

- [x] **ERR-01**: When `--json` is active, errors output structured JSON to stderr
- [ ] **ERR-02**: Non-zero exit codes are preserved for error cases under `--json`

### Compatibility

- [ ] **COMPAT-01**: Default output (no `--json`) remains unchanged for all commands
- [ ] **COMPAT-02**: `actions do --json` flag collision is resolved before adding output `--json` to actions commands

### Commands

- [ ] **CMD-01**: `ap agents list --json` outputs agent list as JSON array
- [ ] **CMD-02**: `ap agents get --json` outputs agent details as JSON object
- [ ] **CMD-03**: `ap sessions list --json` outputs session list as JSON array
- [ ] **CMD-04**: `ap org --json` outputs org info as JSON object
- [ ] **CMD-05**: `ap workspace list --json` outputs workspace list as JSON array
- [ ] **CMD-06**: `ap integrations list --json` outputs connections as JSON array
- [ ] **CMD-07**: `ap permissions --json` outputs permissions as JSON
- [ ] **CMD-08**: `ap actions list --json` outputs actions as JSON array
- [ ] **CMD-09**: `ap auth status --json` outputs auth status as JSON object

## v2 Requirements

### TUI Commands

- **TUI-01**: `ap spawn --json` creates session and returns JSON (no TUI)
- **TUI-02**: `ap session <ID> --json` returns session details as JSON (no TUI)
- **TUI-03**: `ap spawn --json` supports message submission and polling

### Advanced Output

- **ADV-01**: `--jq` flag for filtering JSON output with jq expressions
- **ADV-02**: Pretty-printed JSON output (indented)
- **ADV-03**: `--format` templates for custom output formatting

## Out of Scope

| Feature | Reason |
|---------|--------|
| Auth command JSON output (login/logout) | Auth involves browser flow, not useful for machine consumption |
| JSONL streaming | Adds complexity, not needed for v1 use cases |
| Non-interactive auth (API tokens) | Separate feature, not part of --json milestone |
| New API commands | Only adding JSON to existing commands |
| JSON schemas | Over-engineering for v1 |
| Auto-detect pipe and switch to JSON | Implicit behavior is confusing; explicit --json flag is better |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| OUT-01 | Phase 1 | Pending |
| OUT-02 | Phase 1 | Complete |
| OUT-03 | Phase 1 | Complete |
| OUT-04 | Phase 1 | Complete |
| OUT-05 | Phase 1 | Pending |
| ERR-01 | Phase 1 | Complete |
| ERR-02 | Phase 1 | Pending |
| COMPAT-01 | Phase 1 | Pending |
| COMPAT-02 | Phase 1 | Pending |
| CMD-01 | Phase 2 | Pending |
| CMD-02 | Phase 2 | Pending |
| CMD-03 | Phase 2 | Pending |
| CMD-04 | Phase 2 | Pending |
| CMD-05 | Phase 2 | Pending |
| CMD-06 | Phase 2 | Pending |
| CMD-07 | Phase 2 | Pending |
| CMD-08 | Phase 2 | Pending |
| CMD-09 | Phase 2 | Pending |

**Coverage:**
- v1 requirements: 18 total
- Mapped to phases: 18
- Unmapped: 0

---
*Requirements defined: 2026-03-09*
*Last updated: 2026-03-09 after roadmap creation*
