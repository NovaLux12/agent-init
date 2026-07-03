# Changelog

## 0.1.1 — 2026-07-03

Post-verifier fixes from the M3 review pass.

**Fixed (CRITICAL):**

- **`--from` round-trip silently dropped `created_at` and `voice`.**
  The Card struct didn't model these two optional schema fields, so
  JSON unmarshaling silently dropped them from any `--from` input.
  A user editing a card with `--from` and `--description` would
  watch the rest of the file vanish. Schema validation passed because
  both fields are optional. Fixed by adding `CreatedAt string` and
  `*Voice` to the Card struct, plus a `Voice` type. Regression test
  `TestGenerateFromExistingPreservesAllSchemaFields`.

**Fixed (MEDIUM):**

- **`--verified-by` did not dedupe.** Mirror of the
  `--capability` dedupe pattern: build a `seen` set from existing
  `verified_by`, skip repeats. Without this, repeating a flag or
  re-applying over an existing card produced duplicate entries that
  directories may render as separate badges.

**Fixed (MINOR):**

- **`jsonField` no longer returns an unreachable error.** The body
  passed in is always JSON we just marshaled ourselves; the error
  return was unreachable and the API implied a failure mode that
  couldn't happen. Now returns just `string`.
- **`--version` is now a pre-parse short-circuit.** Matches the
  Go stdlib convention so `agent-init --version --bogus-flag`
  prints the version rather than exiting with the unknown-flag
  error.
- **"Next steps" hints now print in stdout mode too.** Previously
  they only appeared when writing to a file; the `--output -` branch
  silently skipped them. Refactored to a single `emitHints` helper
  called from both branches.

**Tests:** 13 → 17. New regression tests for created_at + voice
preservation, verified_by dedupe, --version with unknown flags, and
"Next steps" hints in stdout mode.

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