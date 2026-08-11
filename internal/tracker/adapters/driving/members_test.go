package driving

import (
	"bytes"
	"context"
	"path/filepath"
	"sattchel/internal/printer"
	"sattchel/internal/tracker/adapters/driven"
	"sattchel/internal/tracker/core"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func TestMemberCLI(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "tracker.json")
	repo := driven.NewFileStorage(dbPath, nil)
	service := core.NewService(repo)
	v := viper.New()
	cfg, err := LoadConfig(v)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	buf := new(bytes.Buffer)
	writer := printer.NewStyleWriterWithWriters(buf, buf)
	cmd := members(service, cfg, writer)

	// Helper to execute commands and get output
	executeCmd := func(args ...string) (string, error) {
		buf.Reset()
		cmd.SetOut(buf)
		cmd.SetErr(buf)
		cmd.SetArgs(args)
		err := cmd.ExecuteContext(context.Background())
		return stripANSI(buf.String()), err
	}

	// 1. List initially - should be empty
	out, err := executeCmd("list")
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if !strings.Contains(out, "No members found") {
		t.Errorf("expected no members, got: %q", out)
	}

	// 2. Create member
	out, err = executeCmd("create", "Alice", "-e", "alice@example.com")
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if !strings.Contains(out, "Member Alice") || !strings.Contains(out, "created successfully") {
		t.Errorf("unexpected output from create: %q", out)
	}

	// Extract the ID from parenthesized text in the output: Member Alice (ID) created successfully
	startIdx := strings.Index(out, "Alice (")
	if startIdx == -1 {
		t.Fatalf("failed to find member ID in output: %q", out)
	}
	startIdx += len("Alice (")
	endIdx := strings.Index(out[startIdx:], ")")
	if endIdx == -1 {
		t.Fatalf("failed to find member ID matching paren in output: %q", out)
	}
	aliceID := out[startIdx : startIdx+endIdx]

	// 3. Get member
	out, err = executeCmd("get", aliceID)
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if !strings.Contains(out, "Member: Alice") || !strings.Contains(out, "alice@example.com") {
		t.Errorf("unexpected output from get: %q", out)
	}

	// 4. Update member
	out, err = executeCmd("update", aliceID, "--name", "Alice Smith", "--email", "alice.smith@example.com")
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if !strings.Contains(out, "Member Alice Smith") || !strings.Contains(out, "updated successfully") {
		t.Errorf("unexpected output from update: %q", out)
	}

	// 5. List again - should contain updated member
	out, err = executeCmd("list")
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if !strings.Contains(out, "Alice Smith") || !strings.Contains(out, "alice.smith@example.com") {
		t.Errorf("unexpected output from list: %q", out)
	}

	// 6. Delete member
	out, err = executeCmd("delete", aliceID)
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	if !strings.Contains(out, "deleted successfully") {
		t.Errorf("unexpected output from delete: %q", out)
	}

	// 7. Get after delete should fail
	_, err = executeCmd("get", aliceID)
	if err == nil {
		t.Fatal("expected error getting deleted member, got nil")
	}
}

func stripANSI(s string) string {
	var sb strings.Builder
	inEscape := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == 0x1b {
			inEscape = true
			continue
		}
		if inEscape {
			if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') {
				inEscape = false
			}
			continue
		}
		sb.WriteByte(c)
	}
	return sb.String()
}
