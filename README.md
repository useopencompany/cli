# Agent Platform CLI

Command-line interface for [Agent Platform](https://agentplatform.cloud).

## Install

```sh
curl -fsSL https://agentplatform.cloud/install.sh | sh
```

## Usage

```sh
ap --help
```

## Core session workflows

Authenticate once:

```sh
ap auth login
```

Start a new Operator session:

```sh
ap spawn
```

List resumable sessions:

```sh
ap sessions
```

Resume a previous session:

```sh
ap session <ID>
```

When resuming older sessions, runtime recovery can happen automatically on turn execution. If key material is missing, the TUI prompts for an Anthropic key and retries the pending turn.

## Docs

Full documentation at [docs.agentplatform.cloud](https://docs.agentplatform.cloud).
