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

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func TestNonInteractiveModeErrors(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "tracker.json")
	repo := driven.NewFileStorage(dbPath, nil)
	service := core.NewService(repo)
	v := viper.New()
	configPath := filepath.Join(tempDir, "config.yml")
	v.SetConfigFile(configPath)
	v.Set("tracker", map[string]any{})
	_ = v.WriteConfigAs(configPath)

	cfg, err := LoadConfig(v)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	writer := &dummyWriter{}

	// Helper to execute any subcommand non-interactively
	execCmd := func(parent *cobra.Command, args ...string) (string, error) {
		var resetFlags func(*cobra.Command)
		resetFlags = func(c *cobra.Command) {
			c.Flags().VisitAll(func(f *pflag.Flag) {
				_ = f.Value.Set(f.DefValue)
			})
			for _, sub := range c.Commands() {
				resetFlags(sub)
			}
		}
		resetFlags(parent)

		buf := new(bytes.Buffer)
		parent.SetOut(buf)
		parent.SetErr(buf)
		parent.SetArgs(args)
		err := parent.ExecuteContext(context.Background())
		return buf.String(), err
	}

	t.Run("projects create without name", func(t *testing.T) {
		cmd := projects(service, cfg, writer)
		_, err := execCmd(cmd, "create")
		if err == nil || !strings.Contains(err.Error(), "project name is required in non-interactive mode") {
			t.Errorf("expected non-interactive project create error, got: %v", err)
		}
	})

	t.Run("projects update without flags", func(t *testing.T) {
		p, err := service.CreateProject(context.Background(), "Update Test", "")
		if err != nil {
			t.Fatalf("failed to create project: %v", err)
		}

		cmd := projects(service, cfg, writer)
		_, err = execCmd(cmd, "update", p.ID)
		if err == nil || !strings.Contains(err.Error(), "at least one flag (--name or --description) must be specified for update in non-interactive mode") {
			t.Errorf("expected non-interactive project update error, got: %v", err)
		}
	})

	t.Run("projects merge without positional args", func(t *testing.T) {
		cmd := projects(service, cfg, writer)
		_, err := execCmd(cmd, "merge")
		if err == nil || !strings.Contains(err.Error(), "both source_project_id and merge_project_id are required in non-interactive mode") {
			t.Errorf("expected non-interactive project merge error, got: %v", err)
		}
	})

	t.Run("projects split without goal ID", func(t *testing.T) {
		p, err := service.CreateProject(context.Background(), "Split Test", "")
		if err != nil {
			t.Fatalf("failed to create project: %v", err)
		}

		cmd := projects(service, cfg, writer)
		_, err = execCmd(cmd, "split", p.ID)
		if err == nil || !strings.Contains(err.Error(), "goal ID (--goal) is required in non-interactive mode") {
			t.Errorf("expected non-interactive project split error, got: %v", err)
		}
	})

	t.Run("projects delete without ID", func(t *testing.T) {
		cmd := projects(service, cfg, writer)
		_, err := execCmd(cmd, "delete")
		if err == nil || !strings.Contains(err.Error(), "project ID is required in non-interactive mode") {
			t.Errorf("expected non-interactive project delete error, got: %v", err)
		}
	})

	t.Run("goals add without name", func(t *testing.T) {
		p, err := service.CreateProject(context.Background(), "Goal Add Test", "")
		if err != nil {
			t.Fatalf("failed to create project: %v", err)
		}
		_ = cfg.SetCurrentProjectID(p.ID)

		buf := new(bytes.Buffer)
		swriter := printer.NewStyleWriterWithWriters(buf, buf)
		cmd := goals(service, cfg, swriter)
		_, err = execCmd(cmd, "add")
		if err == nil || !strings.Contains(err.Error(), "goal name is required in non-interactive mode") {
			t.Errorf("expected non-interactive goals add error, got: %v", err)
		}
	})

	t.Run("goals set non-interactive success and missing ID error", func(t *testing.T) {
		p, err := service.CreateProject(context.Background(), "Goal Set Test", "")
		if err != nil {
			t.Fatalf("failed to create project: %v", err)
		}
		g, err := service.CreateGoal(context.Background(), p.ID, "Root Goal", core.GoalOptions{})
		if err != nil {
			t.Fatalf("failed to create goal: %v", err)
		}
		_ = cfg.SetCurrentProjectID(p.ID)

		buf := new(bytes.Buffer)
		swriter := printer.NewStyleWriterWithWriters(buf, buf)
		cmd := goals(service, cfg, swriter)

		// 1. Missing ID should error
		_, err = execCmd(cmd, "set")
		if err == nil || !strings.Contains(err.Error(), "goal ID is required in non-interactive mode") {
			t.Errorf("expected non-interactive goals set error, got: %v", err)
		}

		// 2. Supplying ID non-interactively should succeed
		_, err = execCmd(cmd, "set", g.ID)
		if err != nil {
			t.Fatalf("goals set with ID failed: %v", err)
		}
		if cfg.CurrentGoalID() != g.ID {
			t.Errorf("expected active goal to be %s, got %s", g.ID, cfg.CurrentGoalID())
		}
	})

	t.Run("goals move without positional args", func(t *testing.T) {
		p, err := service.CreateProject(context.Background(), "Goal Move Test", "")
		if err != nil {
			t.Fatalf("failed to create project: %v", err)
		}
		_ = cfg.SetCurrentProjectID(p.ID)

		buf := new(bytes.Buffer)
		swriter := printer.NewStyleWriterWithWriters(buf, buf)
		cmd := goals(service, cfg, swriter)

		_, err = execCmd(cmd, "move")
		if err == nil || !strings.Contains(err.Error(), "both childId and newParentId are required in non-interactive mode") {
			t.Errorf("expected non-interactive goals move error, got: %v", err)
		}
	})

	t.Run("goals view without ID", func(t *testing.T) {
		p, err := service.CreateProject(context.Background(), "Goal View Test", "")
		if err != nil {
			t.Fatalf("failed to create project: %v", err)
		}
		_ = cfg.SetCurrentProjectID(p.ID)

		buf := new(bytes.Buffer)
		swriter := printer.NewStyleWriterWithWriters(buf, buf)
		cmd := goals(service, cfg, swriter)

		_, err = execCmd(cmd, "view")
		if err == nil || !strings.Contains(err.Error(), "goal ID is required in non-interactive mode") {
			t.Errorf("expected non-interactive goals view error, got: %v", err)
		}
	})

	t.Run("goals update without flags", func(t *testing.T) {
		p, err := service.CreateProject(context.Background(), "Goal Update Test", "")
		if err != nil {
			t.Fatalf("failed to create project: %v", err)
		}
		g, err := service.CreateGoal(context.Background(), p.ID, "Root Goal", core.GoalOptions{})
		if err != nil {
			t.Fatalf("failed to create goal: %v", err)
		}
		_ = cfg.SetCurrentProjectID(p.ID)

		buf := new(bytes.Buffer)
		swriter := printer.NewStyleWriterWithWriters(buf, buf)
		cmd := goals(service, cfg, swriter)

		_, err = execCmd(cmd, "update", g.ID)
		if err == nil || !strings.Contains(err.Error(), "at least one flag must be specified for update in non-interactive mode") {
			t.Errorf("expected non-interactive goals update error, got: %v", err)
		}
	})

	t.Run("member create without name", func(t *testing.T) {
		cmd := members(service, cfg, writer)
		_, err := execCmd(cmd, "create")
		if err == nil || !strings.Contains(err.Error(), "member name is required in non-interactive mode") {
			t.Errorf("expected non-interactive member create error, got: %v", err)
		}
	})

	t.Run("member get without ID", func(t *testing.T) {
		cmd := members(service, cfg, writer)
		_, err := execCmd(cmd, "get")
		if err == nil || !strings.Contains(err.Error(), "member ID is required in non-interactive mode") {
			t.Errorf("expected non-interactive member get error, got: %v", err)
		}
	})

	t.Run("member update without flags", func(t *testing.T) {
		m, err := service.CreateMember(context.Background(), "Alice", "alice@example.com")
		if err != nil {
			t.Fatalf("failed to create member: %v", err)
		}

		cmd := members(service, cfg, writer)
		_, err = execCmd(cmd, "update", m.ID)
		if err == nil || !strings.Contains(err.Error(), "at least one of --name or --email must be specified for update in non-interactive mode") {
			t.Errorf("expected non-interactive member update error, got: %v", err)
		}
	})

	t.Run("member delete without ID", func(t *testing.T) {
		cmd := members(service, cfg, writer)
		_, err := execCmd(cmd, "delete")
		if err == nil || !strings.Contains(err.Error(), "member ID is required in non-interactive mode") {
			t.Errorf("expected non-interactive member delete error, got: %v", err)
		}
	})
}
