# agent-init

Interactive Go CLI to generate `agent.json` identity cards against the
[reflectt/agent-identity-kit](https://github.com/reflectt/agent-identity-kit)
v1 schema. The "create" half of the create-and-validate pair
([`agent-validate`](../agent-validate) is the "check" half).

```text
$ agent-init \
    --name "My Agent" \
    --handle "@me@example.com" \
    --description "What this agent does." \
    --owner-name "Example Inc" \
    --capability code-generation \
    --capability web-search \
    --runtime openclaw \
    --card-url "https://example.com/.well-known/agent.json" \
    --output ./agent.json

wrote 587 bytes to ./agent.json
  schema validation: PASS

Next steps:
  - Validate: agent-validate ./agent.json
  - Host at: https://example.com/.well-known/agent.json
```

## Why this exists

The upstream `reflectt/agent-identity-kit` ships `init.sh`, a bash
wrapper that prompts interactively and uses heredoc templating. It
works but has the same downsides as the upstream `validate.sh` —
needs a shell, can't be embedded in Go tooling, no flag-driven mode
for CI.

`agent-init` is the same job in a single ~10 MB static binary with
no runtime dependencies. It also **validates the output as it
writes**, so you can't accidentally publish an invalid card —
the file is only written if it passes schema validation.

## Install

```sh
go install github.com/NovaLux12/agent-init@latest
```

Or grab a binary from
[Releases](https://github.com/NovaLux12/agent-init/releases).

## Usage

### Minimal card

```sh
agent-init \
    --name "My Agent" \
    --handle "@me@example.com" \
    --description "What this agent does." \
    --owner-name "My Name"
```

This produces a minimal valid `agent.json` (only the required fields
plus `updated_at`). Sufficient for directories to accept the card.

### Full card

```sh
agent-init \
    --name "My Agent" \
    --handle "@me@example.com" \
    --description "What this agent does." \
    --owner-name "My Org" \
    --owner-url "https://example.com" \
    --owner-contact "team@example.com" \
    --runtime openclaw \
    --model "minimax/MiniMax-M3" \
    --platform-version "1.0.0" \
    --avatar "https://example.com/avatar.png" \
    --homepage "https://example.com/agent" \
    --capability code-generation \
    --capability web-search \
    --capability file-operations \
    --mcp \
    --card-url "https://example.com/.well-known/agent.json" \
    --inbox-url "https://example.com/inbox" \
    --trust-level active \
    --verified-by foragents.dev \
    --link-website "https://example.com" \
    --link-repo "https://github.com/example/agent" \
    --output ./agent.json
```

### Edit an existing card

```sh
agent-init --from ./agent.json --description "Updated description." --output ./agent.json
```

The `--from` card is loaded first, then any flags overlay on top.
Unspecified fields keep their existing values. Capabilities passed
via `--capability` are *appended* (with deduplication), not
replaced.

### To stdout

```sh
agent-init --name X --handle @x@x.com --description x --owner-name O --output -
```

### Validate-after-generate

The CLI runs schema validation before writing. If the generated
card would be invalid, no file is written and the errors are
printed to stderr. Common causes:

- Missing `--name`, `--handle`, `--description`, or `--owner-name`
- Handle in wrong format (must be `@name@domain`)
- Description over 500 characters (schema limit)

## Flags

| Flag | Purpose |
|------|---------|
| `--from FILE` | start from existing agent.json (overlays, not replaces) |
| `--output FILE` | destination file (default `./agent.json`, `-` for stdout) |
| `--name NAME` | agent.name (required) |
| `--handle HANDLE` | agent.handle — `@name@domain` format (required) |
| `--description DESC` | agent.description (required, max 500 chars) |
| `--avatar URL` | agent.avatar URL |
| `--homepage URL` | agent.homepage URL |
| `--owner-name NAME` | owner.name (required) |
| `--owner-url URL` | owner.url |
| `--owner-contact EMAIL` | owner.contact |
| `--owner-verified` | owner.verified=true (use only if a registry vouches) |
| `--runtime RUNTIME` | platform.runtime (e.g. openclaw) |
| `--model MODEL` | platform.model |
| `--platform-version V` | platform.version |
| `--capability TAG` | add a capability tag (repeatable) |
| `--mcp` | protocols.mcp=true |
| `--a2a` | protocols.a2a=true |
| `--protocol-http` | protocols.http=true |
| `--card-url URL` | endpoints.card — canonical URL for this card |
| `--inbox-url URL` | endpoints.inbox |
| `--status-url URL` | endpoints.status |
| `--trust-level LEVEL` | trust.level (`new`/`active`/`established`/`verified`) |
| `--verified-by NAME` | add a trust.verified_by entry (repeatable) |
| `--link-website URL` | links.website |
| `--link-repo URL` | links.repo |
| `--link-docs URL` | links.documentation |
| `--now` | always stamp `updated_at` to current time |
| `--version` | print version and exit |

## Exit codes

| code | meaning |
| ---: | :------- |
|    0 | success |
|    3 | generation failed (validation or write) |
|    4 | argument error |

## Use as a library

```go
import "github.com/NovaLux12/agent-init"

body, err := agentinit.Generate(ctx, agentinit.Options{
    AgentName:        "My Agent",
    AgentHandle:      "@me@example.com",
    AgentDescription: "What this agent does.",
    OwnerName:        "My Org",
    Capabilities:     []string{"code-generation"},
})
```

The `Generate` function returns the marshaled card bytes (with a
trailing newline) and validates via
[`agent-validate`](https://github.com/NovaLux12/agent-validate) before
returning. If validation fails, no bytes are returned and the error
explains why.

## Related

- [`NovaLux12/agent-validate`](../agent-validate) — schema validator
- [`NovaLux12/agentcard-mcp`](../agentcard-mcp) — MCP server over
  the same cards
- [`reflectt/agent-identity-kit`](https://github.com/reflectt/agent-identity-kit)
  — the spec

## License

MIT.