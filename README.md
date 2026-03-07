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

## Organization and integrations

```sh
ap org
ap org invite --email teammate@example.com

ap workspace list
ap workspace create --name "Product"
ap workspace switch <WORKSPACE_ID>

ap integrations connect --integration slack --scope user_private_workspace --token xoxb-...
ap integrations list
ap integrations revoke <CONNECTION_ID>

ap actions
ap find slack
ap do slack.channels.list --input limit=20
ap actions invocations

ap dashboard
```

## Docs

Full documentation at [docs.agentplatform.cloud](https://docs.agentplatform.cloud).
