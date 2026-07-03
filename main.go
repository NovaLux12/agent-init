// Command agent-init generates agent.json identity cards against the
// reflectt/agent-identity-kit v1 schema. It is the "create" half of
// the create-and-validate pair; agent-validate is the "check" half.
//
// Usage (non-interactive, flag-driven):
//
//	agent-init --name "My Agent" \
//	           --handle "@me@example.com" \
//	           --description "What this agent does." \
//	           --owner-name "Example Inc" \
//	           --owner-url "https://example.com" \
//	           --capability code-generation \
//	           --capability web-search \
//	           --runtime openclaw \
//	           --card-url "https://example.com/.well-known/agent.json" \
//	           --output ./agent.json
//
// Start from an existing card and overlay:
//
//	agent-init --from ./agent.json --name "Updated Name" --output ./agent.json
//
// The output is validated against the embedded v1 schema before being
// written. If validation fails, the file is NOT written and the
// errors are printed.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/NovaLux12/agent-validate/pkg/agentvalidate"
)

// Version is the agent-init release tag. Bump in lockstep with
// CHANGELOG.md and README install snippets.
const Version = "0.1.0"

// FlagCapturer lets the generator use injected flag values for
// testing without going through os.Args. The default is real os.Args.
type FlagCapturer func(name string) (string, bool)

// Card is the shape we build up in memory before writing. It mirrors
// the schema's top-level fields so callers can mix-and-match what
// they set vs. what they leave to defaults.
type Card struct {
	Version      string     `json:"version"`
	Agent        Agent      `json:"agent"`
	Owner        Owner      `json:"owner"`
	Platform     *Platform  `json:"platform,omitempty"`
	Capabilities []string   `json:"capabilities,omitempty"`
	Protocols    *Protocols `json:"protocols,omitempty"`
	Endpoints    *Endpoints `json:"endpoints,omitempty"`
	Trust        *Trust     `json:"trust,omitempty"`
	Links        *Links     `json:"links,omitempty"`
	UpdatedAt    string     `json:"updated_at,omitempty"`
}

type Agent struct {
	Name        string `json:"name"`
	Handle      string `json:"handle"`
	Description string `json:"description"`
	Avatar      string `json:"avatar,omitempty"`
	Homepage    string `json:"homepage,omitempty"`
}

type Owner struct {
	Name     string `json:"name"`
	URL      string `json:"url,omitempty"`
	Contact  string `json:"contact,omitempty"`
	Verified bool   `json:"verified,omitempty"`
}

type Platform struct {
	Runtime string `json:"runtime,omitempty"`
	Model   string `json:"model,omitempty"`
	Version string `json:"version,omitempty"`
}

type Protocols struct {
	MCP       bool   `json:"mcp,omitempty"`
	A2A       bool   `json:"a2a,omitempty"`
	AgentCard string `json:"agent-card,omitempty"`
	HTTP      bool   `json:"http,omitempty"`
}

type Endpoints struct {
	Card   string `json:"card,omitempty"`
	Inbox  string `json:"inbox,omitempty"`
	Status string `json:"status,omitempty"`
}

type Trust struct {
	Level        string   `json:"level"`
	Created      string   `json:"created,omitempty"`
	VerifiedBy   []string `json:"verified_by,omitempty"`
	Attestations []any    `json:"attestations,omitempty"`
}

type Links struct {
	Website       string   `json:"website,omitempty"`
	Repo          string   `json:"repo,omitempty"`
	Documentation string   `json:"documentation,omitempty"`
	Social        []Social `json:"social,omitempty"`
}

type Social struct {
	Platform string `json:"platform"`
	URL      string `json:"url"`
}

// Options configures a single Generate call. It is the struct form of
// the CLI flags and can be constructed directly for testing.
type Options struct {
	// From is an optional path to an existing card to start from.
	From string
	// Output is the destination file path. Empty means stdout.
	Output string
	// NonInteractive disables any prompt-style flow; required when
	// any required field is missing.
	NonInteractive bool
	// Agent fields
	AgentName        string
	AgentHandle      string
	AgentDescription string
	AgentAvatar      string
	AgentHomepage    string
	// Owner fields
	OwnerName     string
	OwnerURL      string
	OwnerContact  string
	OwnerVerified bool
	// Platform fields
	PlatformRuntime string
	PlatformModel   string
	PlatformVersion string
	// Capabilities — flag may be specified multiple times.
	Capabilities []string
	// Protocols
	ProtocolMCP  bool
	ProtocolA2A  bool
	ProtocolHTTP bool
	// Endpoints
	EndpointCard   string
	EndpointInbox  string
	EndpointStatus string
	// Trust
	TrustLevel      string
	TrustVerifiedBy []string
	// Links
	LinkWebsite       string
	LinkRepo          string
	LinkDocumentation string
	// Now sets updated_at; defaults to time.Now().UTC() when true.
	Now bool
}

// Generate builds a Card from Options, optionally starting from an
// existing card, validates it, and returns the JSON bytes (with a
// trailing newline). It does NOT write to disk — the caller decides.
func Generate(ctx context.Context, opts Options) ([]byte, error) {
	card := Card{Version: "1.0"}

	// If --from is set, overlay the existing card first.
	if opts.From != "" {
		data, err := os.ReadFile(opts.From)
		if err != nil {
			return nil, fmt.Errorf("could not read --from %s: %w", opts.From, err)
		}
		if err := json.Unmarshal(data, &card); err != nil {
			return nil, fmt.Errorf("--from %s is not valid JSON: %w", opts.From, err)
		}
		if card.Version == "" {
			card.Version = "1.0"
		}
	}

	// Overlay fields from opts. Zero-value means "don't change".
	if opts.AgentName != "" {
		card.Agent.Name = opts.AgentName
	}
	if opts.AgentHandle != "" {
		card.Agent.Handle = opts.AgentHandle
	}
	if opts.AgentDescription != "" {
		card.Agent.Description = opts.AgentDescription
	}
	if opts.AgentAvatar != "" {
		card.Agent.Avatar = opts.AgentAvatar
	}
	if opts.AgentHomepage != "" {
		card.Agent.Homepage = opts.AgentHomepage
	}

	if opts.OwnerName != "" {
		card.Owner.Name = opts.OwnerName
	}
	if opts.OwnerURL != "" {
		card.Owner.URL = opts.OwnerURL
	}
	if opts.OwnerContact != "" {
		card.Owner.Contact = opts.OwnerContact
	}
	if opts.OwnerVerified {
		card.Owner.Verified = true
	}

	// Platform is an optional object — only create it if any sub-field is set.
	if opts.PlatformRuntime != "" || opts.PlatformModel != "" || opts.PlatformVersion != "" {
		if card.Platform == nil {
			card.Platform = &Platform{}
		}
		if opts.PlatformRuntime != "" {
			card.Platform.Runtime = opts.PlatformRuntime
		}
		if opts.PlatformModel != "" {
			card.Platform.Model = opts.PlatformModel
		}
		if opts.PlatformVersion != "" {
			card.Platform.Version = opts.PlatformVersion
		}
	}

	if len(opts.Capabilities) > 0 {
		// Append to existing capabilities if the user passed --from.
		existing := make(map[string]struct{}, len(card.Capabilities))
		for _, c := range card.Capabilities {
			existing[c] = struct{}{}
		}
		for _, c := range opts.Capabilities {
			if _, dup := existing[c]; dup {
				continue
			}
			card.Capabilities = append(card.Capabilities, c)
			existing[c] = struct{}{}
		}
	}

	if opts.ProtocolMCP || opts.ProtocolA2A || opts.ProtocolHTTP {
		if card.Protocols == nil {
			card.Protocols = &Protocols{AgentCard: "1.0"}
		}
		if opts.ProtocolMCP {
			card.Protocols.MCP = true
		}
		if opts.ProtocolA2A {
			card.Protocols.A2A = true
		}
		if opts.ProtocolHTTP {
			card.Protocols.HTTP = true
		}
	}

	if opts.EndpointCard != "" || opts.EndpointInbox != "" || opts.EndpointStatus != "" {
		if card.Endpoints == nil {
			card.Endpoints = &Endpoints{}
		}
		if opts.EndpointCard != "" {
			card.Endpoints.Card = opts.EndpointCard
		}
		if opts.EndpointInbox != "" {
			card.Endpoints.Inbox = opts.EndpointInbox
		}
		if opts.EndpointStatus != "" {
			card.Endpoints.Status = opts.EndpointStatus
		}
	}

	if opts.TrustLevel != "" || len(opts.TrustVerifiedBy) > 0 {
		if card.Trust == nil {
			card.Trust = &Trust{Level: "new"}
		}
		if opts.TrustLevel != "" {
			card.Trust.Level = opts.TrustLevel
		}
		if len(opts.TrustVerifiedBy) > 0 {
			card.Trust.VerifiedBy = append(card.Trust.VerifiedBy, opts.TrustVerifiedBy...)
		}
	}

	if opts.LinkWebsite != "" || opts.LinkRepo != "" || opts.LinkDocumentation != "" {
		if card.Links == nil {
			card.Links = &Links{}
		}
		if opts.LinkWebsite != "" {
			card.Links.Website = opts.LinkWebsite
		}
		if opts.LinkRepo != "" {
			card.Links.Repo = opts.LinkRepo
		}
		if opts.LinkDocumentation != "" {
			card.Links.Documentation = opts.LinkDocumentation
		}
	}

	// Validate before rendering. This catches missing required fields
	// and tells the caller exactly what to fix before the file is
	// written.
	results, err := agentvalidate.Validate(ctx, marshaledCard(card))
	if err != nil {
		return nil, fmt.Errorf("schema validation could not run: %w", err)
	}
	if len(results) > 0 {
		return nil, fmt.Errorf("card is not valid against v1 schema:\n  - %s",
			joinResults(results))
	}

	// Always stamp updated_at so the generated card doesn't show up
	// in directory listings as stale-by-default.
	now := time.Now().UTC().Format(time.RFC3339)
	if opts.Now {
		card.UpdatedAt = now
	} else if card.UpdatedAt == "" {
		card.UpdatedAt = now
	}

	body, err := json.MarshalIndent(card, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("could not marshal card: %w", err)
	}
	return append(body, '\n'), nil
}

// joinResults renders schema validation errors one per line for
// clean stderr output.
func joinResults(results []agentvalidate.Result) string {
	var b strings.Builder
	for i, r := range results {
		if i > 0 {
			b.WriteString("\n  - ")
		} else {
			b.WriteString("- ")
		}
		b.WriteString(r.String())
	}
	return b.String()
}

// marshaledCard produces a marshaled JSON form of c for use with
// the validator. We need this in Generate so we can validate before
// the final pretty-print step.
func marshaledCard(c Card) []byte {
	b, _ := json.Marshal(c)
	return b
}

// runCLI parses os.Args, invokes Generate, and either writes to a
// file or prints to stdout. Returns the process exit code.
//
// stdout and stderr are io.Writer rather than *os.File so tests can
// inject capture buffers.
func runCLI(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("agent-init", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var opts Options
	fs.StringVar(&opts.From, "from", "", "path to an existing agent.json to overlay fields onto")
	fs.StringVar(&opts.Output, "output", "", "destination file path (default: ./agent.json; use '-' for stdout)")
	fs.BoolVar(&opts.NonInteractive, "non-interactive", true, "fail if any required flag is missing (default true; --interactive is reserved for future use)")

	fs.StringVar(&opts.AgentName, "name", "", "agent.display name (required: agent.name)")
	fs.StringVar(&opts.AgentHandle, "handle", "", "Fediverse-style handle @name@domain (required: agent.handle)")
	fs.StringVar(&opts.AgentDescription, "description", "", "what this agent does (required: agent.description)")
	fs.StringVar(&opts.AgentAvatar, "avatar", "", "URL to agent's avatar image")
	fs.StringVar(&opts.AgentHomepage, "homepage", "", "URL to agent's homepage")

	fs.StringVar(&opts.OwnerName, "owner-name", "", "owner's display name (required: owner.name)")
	fs.StringVar(&opts.OwnerURL, "owner-url", "", "owner's website URL")
	fs.StringVar(&opts.OwnerContact, "owner-contact", "", "owner's contact email or URL")
	fs.BoolVar(&opts.OwnerVerified, "owner-verified", false, "set owner.verified=true (only if a registry actually vouches)")

	fs.StringVar(&opts.PlatformRuntime, "runtime", "", "platform.runtime (e.g. openclaw)")
	fs.StringVar(&opts.PlatformModel, "model", "", "platform.model (e.g. claude-sonnet-4-20250514)")
	fs.StringVar(&opts.PlatformVersion, "platform-version", "", "platform.version (e.g. 1.2.0)")

	fs.Func("capability", "add a capability tag (repeatable)", func(s string) error {
		opts.Capabilities = append(opts.Capabilities, s)
		return nil
	})

	fs.BoolVar(&opts.ProtocolMCP, "mcp", false, "set protocols.mcp=true")
	fs.BoolVar(&opts.ProtocolA2A, "a2a", false, "set protocols.a2a=true")
	fs.BoolVar(&opts.ProtocolHTTP, "protocol-http", false, "set protocols.http=true")

	fs.StringVar(&opts.EndpointCard, "card-url", "", "endpoints.card (canonical URL for this card)")
	fs.StringVar(&opts.EndpointInbox, "inbox-url", "", "endpoints.inbox")
	fs.StringVar(&opts.EndpointStatus, "status-url", "", "endpoints.status")

	fs.StringVar(&opts.TrustLevel, "trust-level", "", "trust.level (new|active|established|verified)")
	fs.Func("verified-by", "add a verified_by entry (repeatable)", func(s string) error {
		opts.TrustVerifiedBy = append(opts.TrustVerifiedBy, s)
		return nil
	})

	fs.StringVar(&opts.LinkWebsite, "link-website", "", "links.website")
	fs.StringVar(&opts.LinkRepo, "link-repo", "", "links.repo")
	fs.StringVar(&opts.LinkDocumentation, "link-docs", "", "links.documentation")

	fs.BoolVar(&opts.Now, "now", false, "always stamp updated_at to the current time, even when --from has one")

	versionFlag := fs.Bool("version", false, "print version and exit")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 4
	}

	if *versionFlag {
		fmt.Fprintf(stdout, "agent-init %s\n", Version)
		return 0
	}

	// Default output is ./agent.json unless explicitly '-'.
	if opts.Output == "" {
		opts.Output = "agent.json"
	}

	ctx := context.Background()
	body, err := Generate(ctx, opts)
	if err != nil {
		fmt.Fprintf(stderr, "error: %s\n", err)
		return 3
	}

	if opts.Output == "-" {
		stdout.Write(body)
		return 0
	}
	if err := os.WriteFile(opts.Output, body, 0o644); err != nil {
		fmt.Fprintf(stderr, "error: could not write %s: %v\n", opts.Output, err)
		return 3
	}
	fmt.Fprintf(stdout, "wrote %d bytes to %s\n", len(body), opts.Output)
	fmt.Fprintf(stdout, "  schema validation: PASS\n")
	fmt.Fprintf(stdout, "\nNext steps:\n")
	fmt.Fprintf(stdout, "  - Validate: agent-validate %s\n", opts.Output)
	if cardURL, _ := jsonField(body, "endpoints", "card"); cardURL != "" {
		fmt.Fprintf(stdout, "  - Host at: %s\n", cardURL)
	} else {
		fmt.Fprintf(stdout, "  - Host at: <your-domain>/.well-known/agent.json\n")
	}
	return 0
}

func main() {
	os.Exit(runCLI(os.Args[1:], os.Stdout, os.Stderr))
}

// jsonField reads a nested string field from raw JSON. Returns ""
// if the path is missing or wrong type.
func jsonField(body []byte, keys ...string) (string, error) {
	var v any
	if err := json.Unmarshal(body, &v); err != nil {
		return "", err
	}
	cur := v
	for _, k := range keys {
		m, ok := cur.(map[string]any)
		if !ok {
			return "", nil
		}
		cur = m[k]
	}
	s, _ := cur.(string)
	return s, nil
}
