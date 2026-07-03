# Changelog

## 0.1.0 — 2026-07-03

First public release. Flag-driven Go CLI that generates
`agent.json` identity cards against the
[`reflectt/agent-identity-kit`](https://github.com/reflectt/agent-identity-kit)
v1 schema, validates the output before writing, and supports
overlay-on-existing via `--from`.

Pairs with [`agent-validate`](../agent-validate) — same schema,
same library, complementary halves of the create-and-validate
workflow.

**Features:**

- Required fields: `--name`, `--handle`, `--description`,
  `--owner-name`. Optional fields cover platform, capabilities,
  protocols, endpoints, trust, links.
- `--capability` is repeatable and deduplicates against any
  existing values when used with `--from`.
- `--from <path>` overlays new flags on top of an existing card.
- Validates output against the embedded v1 schema before writing —
  no invalid file is ever produced.
- `--output -` writes to stdout for shell pipelines.
- Single static binary, zero runtime deps.

**Tests:** 13 across unit (overlay behavior, dedupe, validation
feedback, missing fields, invalid --from JSON) and end-to-end
CLI (--version, --output -).

**Compatibility:** Linux, macOS, Windows × amd64, arm64 via Go
cross-compilation.