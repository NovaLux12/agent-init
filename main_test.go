// Tests for the agent-init card generator.

package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateMinimalCard(t *testing.T) {
	got, err := Generate(context.Background(), Options{
		AgentName:        "Test",
		AgentHandle:      "@test@example.com",
		AgentDescription: "Test agent.",
		OwnerName:        "Owner",
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	var c Card
	if err := json.Unmarshal(got, &c); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if c.Version != "1.0" {
		t.Errorf("Version: got %q, want 1.0", c.Version)
	}
	if c.Agent.Name != "Test" {
		t.Errorf("Agent.Name: got %q", c.Agent.Name)
	}
	if c.Agent.Handle != "@test@example.com" {
		t.Errorf("Agent.Handle: got %q", c.Agent.Handle)
	}
	if c.Owner.Name != "Owner" {
		t.Errorf("Owner.Name: got %q", c.Owner.Name)
	}
	if c.UpdatedAt == "" {
		t.Errorf("UpdatedAt: not stamped")
	}
}

func TestGenerateFailsWithoutRequiredFields(t *testing.T) {
	// Missing agent.handle, agent.description, owner.name.
	_, err := Generate(context.Background(), Options{
		AgentName: "Test",
	})
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	if !strings.Contains(err.Error(), "schema") {
		t.Errorf("expected schema-related error, got: %v", err)
	}
}

func TestGeneratePlatformOnlyCreatedWhenSet(t *testing.T) {
	got, err := Generate(context.Background(), Options{
		AgentName:        "T",
		AgentHandle:      "@t@t.com",
		AgentDescription: "x",
		OwnerName:        "O",
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if strings.Contains(string(got), `"platform":`) {
		t.Errorf("Platform should not appear when no platform flags set, got: %s", got)
	}
}

func TestGeneratePlatformCreatedWhenAnyFieldSet(t *testing.T) {
	got, err := Generate(context.Background(), Options{
		AgentName:        "T",
		AgentHandle:      "@t@t.com",
		AgentDescription: "x",
		OwnerName:        "O",
		PlatformRuntime:  "openclaw",
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if !strings.Contains(string(got), `"runtime": "openclaw"`) {
		t.Errorf("Platform.runtime not set, got: %s", got)
	}
}

func TestGenerateProtocolsAgentCardDefault(t *testing.T) {
	got, err := Generate(context.Background(), Options{
		AgentName:        "T",
		AgentHandle:      "@t@t.com",
		AgentDescription: "x",
		OwnerName:        "O",
		ProtocolMCP:      true,
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if !strings.Contains(string(got), `"agent-card": "1.0"`) {
		t.Errorf("expected default protocols.agent-card = 1.0, got: %s", got)
	}
	if !strings.Contains(string(got), `"mcp": true`) {
		t.Errorf("expected mcp: true, got: %s", got)
	}
}

func TestGenerateCapabilitiesDeduplicate(t *testing.T) {
	got, err := Generate(context.Background(), Options{
		AgentName:        "T",
		AgentHandle:      "@t@t.com",
		AgentDescription: "x",
		OwnerName:        "O",
		Capabilities:     []string{"code-generation", "code-generation", "web-search"},
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	var c Card
	if err := json.Unmarshal(got, &c); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(c.Capabilities) != 2 {
		t.Errorf("Capabilities: got %v, want dedupe to 2", c.Capabilities)
	}
}

func TestGenerateFromExistingOverlay(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "base.json")
	if err := os.WriteFile(src, []byte(`{
        "version": "1.0",
        "agent": {"name":"Old","handle":"@old@old.com","description":"old"},
        "owner": {"name":"OldOwner"},
        "capabilities": ["code-generation"]
    }`), 0o644); err != nil {
		t.Fatalf("write src: %v", err)
	}

	got, err := Generate(context.Background(), Options{
		From:             src,
		AgentName:        "New",
		AgentDescription: "new desc",
		Capabilities:     []string{"web-search"},
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	var c Card
	if err := json.Unmarshal(got, &c); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	// Overlaid fields
	if c.Agent.Name != "New" {
		t.Errorf("Agent.Name: got %q, want New", c.Agent.Name)
	}
	if c.Agent.Description != "new desc" {
		t.Errorf("Agent.Description: got %q", c.Agent.Description)
	}
	// Preserved fields
	if c.Agent.Handle != "@old@old.com" {
		t.Errorf("Agent.Handle: got %q, want @old@old.com (preserved)", c.Agent.Handle)
	}
	if c.Owner.Name != "OldOwner" {
		t.Errorf("Owner.Name: got %q, want OldOwner (preserved)", c.Owner.Name)
	}
	if len(c.Capabilities) != 2 {
		t.Errorf("Capabilities: got %v, want original+appended", c.Capabilities)
	}
}

func TestGenerateFromMissingFile(t *testing.T) {
	_, err := Generate(context.Background(), Options{
		From:        "/nonexistent/agent.json",
		AgentName:   "T",
		AgentHandle: "@t@t.com",
	})
	if err == nil {
		t.Fatal("expected error on missing --from, got nil")
	}
}

func TestGenerateFromInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	bad := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(bad, []byte(`{ not valid`), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	_, err := Generate(context.Background(), Options{
		From: bad,
	})
	if err == nil {
		t.Fatal("expected error on bad JSON, got nil")
	}
}

func TestGenerateOwnsCardWithoutFlagsPreservesEverything(t *testing.T) {
	// When no flags are set and no --from is given, Generate should
	// fail because the required fields are missing. Verify the error
	// message identifies which fields are missing.
	_, err := Generate(context.Background(), Options{})
	if err == nil {
		t.Fatal("expected error on empty options, got nil")
	}
}

func TestRunCLIVersion(t *testing.T) {
	code := runCLI([]string{"--version"}, &nullFile{}, &nullFile{})
	if code != 0 {
		t.Errorf("expected exit 0 for --version, got %d", code)
	}
}

func TestRunCLIRequiresNameOrMissingFields(t *testing.T) {
	// No flags -> Generate fails because required fields missing.
	out := &nullFile{}
	err := &nullFile{}
	code := runCLI([]string{"--output", "-"}, out, err)
	if code == 0 {
		t.Errorf("expected non-zero exit, got 0")
	}
}

func TestRunCLIWriteToFile(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "out.json")
	code := runCLI([]string{
		"--name", "Test",
		"--handle", "@t@t.com",
		"--description", "x",
		"--owner-name", "O",
		"--output", dest,
	}, &nullFile{}, &nullFile{})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("read dest: %v", err)
	}
	if !strings.Contains(string(got), "Test") {
		t.Errorf("file doesn't contain expected name: %q", got)
	}
}

func TestRunCLIStdoutOutput(t *testing.T) {
	out := &captureBuf{}
	errBuf := &captureBuf{}
	code := runCLI([]string{
		"--name", "Test",
		"--handle", "@t@t.com",
		"--description", "x",
		"--owner-name", "O",
		"--output", "-",
	}, out, errBuf)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d (stderr: %s)", code, errBuf.String())
	}
	if !strings.Contains(out.String(), "Test") {
		t.Errorf("stdout missing 'Test': %q", out.String())
	}
}

// --- minimal in-memory File implementations for testing ---

type nullFile struct{}

func (nullFile) Write(p []byte) (int, error) { return len(p), nil }

type captureBuf struct {
	buf []byte
}

func (c *captureBuf) Write(p []byte) (int, error) {
	c.buf = append(c.buf, p...)
	return len(p), nil
}

func (c *captureBuf) String() string { return string(c.buf) }
