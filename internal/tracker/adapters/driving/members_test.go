package driving

import (
	"bytes"
	"context"
	"path/filepath"
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

	cmd := members(service, cfg)

	// Helper to execute commands and get output
	executeCmd := func(args ...string) (string, error) {
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetErr(buf)
		cmd.SetArgs(args)
		err := cmd.ExecuteContext(context.Background())
		return buf.String(), err
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
	if !strings.Contains(out, "Member created successfully") || !strings.Contains(out, "Name: Alice") || !strings.Contains(out, "Email: alice@example.com") {
		t.Errorf("unexpected output from create: %q", out)
	}

	// Extract the ID from the output
	lines := strings.Split(out, "\n")
	var aliceID string
	for _, line := range lines {
		if strings.HasPrefix(line, "ID: ") {
			aliceID = strings.TrimPrefix(line, "ID: ")
			break
		}
	}
	if aliceID == "" {
		t.Fatalf("failed to find member ID in output: %q", out)
	}

	// 3. Get member
	out, err = executeCmd("get", aliceID)
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if !strings.Contains(out, "Name: Alice") || !strings.Contains(out, "Email: alice@example.com") {
		t.Errorf("unexpected output from get: %q", out)
	}

	// 4. Update member
	out, err = executeCmd("update", aliceID, "--name", "Alice Smith", "--email", "alice.smith@example.com")
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if !strings.Contains(out, "Member updated successfully") || !strings.Contains(out, "Name: Alice Smith") || !strings.Contains(out, "Email: alice.smith@example.com") {
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
