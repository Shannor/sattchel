package driving

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"sattchel/internal/printer"
	"sattchel/internal/tracker/adapters/driven"
	"sattchel/internal/tracker/core"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func TestGoalsTriageSkipsCompletedGoals(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "tracker.json")
	repo := driven.NewFileStorage(dbPath, nil)
	service := core.NewService(repo)
	v := viper.New()
	cfg, err := LoadConfig(v)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	project, err := service.CreateProject(context.Background(), "Proj A", "")
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	rootGoal, err := service.CreateGoal(context.Background(), project.ID, "Root Goal", core.GoalOptions{})
	if err != nil {
		t.Fatalf("failed to create root goal: %v", err)
	}

	_, err = service.CreateGoal(context.Background(), project.ID, "Open Do It Now", core.GoalOptions{
		ParentID: rootGoal.ID,
		Status:   core.GoalOpen,
		Impact:   core.HighImpact,
		Effort:   core.LowEffort,
	})
	if err != nil {
		t.Fatalf("failed to create open triage goal: %v", err)
	}

	_, err = service.CreateGoal(context.Background(), project.ID, "Completed Do It Now", core.GoalOptions{
		ParentID: rootGoal.ID,
		Status:   core.GoalCompleted,
		Impact:   core.HighImpact,
		Effort:   core.LowEffort,
	})
	if err != nil {
		t.Fatalf("failed to create completed triage goal: %v", err)
	}

	_, err = service.CreateGoal(context.Background(), project.ID, "Open Missing Member", core.GoalOptions{
		ParentID: rootGoal.ID,
		Status:   core.GoalOpen,
		Impact:   core.MediumImpact,
		Effort:   core.MediumEffort,
	})
	if err != nil {
		t.Fatalf("failed to create open missing goal: %v", err)
	}

	_, err = service.CreateGoal(context.Background(), project.ID, "Completed Missing Member", core.GoalOptions{
		ParentID: rootGoal.ID,
		Status:   core.GoalCompleted,
		Impact:   core.MediumImpact,
		Effort:   core.MediumEffort,
	})
	if err != nil {
		t.Fatalf("failed to create completed missing goal: %v", err)
	}

	writer := printer.NewStyleWriterWithWriters(new(bytes.Buffer), new(bytes.Buffer))

	t.Run("default triage hides completed goals", func(t *testing.T) {
		cmd := triageGoals(service, cfg, writer)
		cmd.SetArgs([]string{"--projectId", project.ID, "--stdout"})

		out := captureStdout(t, func() {
			if err := cmd.ExecuteContext(context.Background()); err != nil {
				t.Fatalf("triage command failed: %v", err)
			}
		})

		if !strings.Contains(out, "Open Do It Now") {
			t.Fatalf("expected open goal in triage output, got: %q", out)
		}
		if strings.Contains(out, "Completed Do It Now") {
			t.Fatalf("did not expect completed triage goal in output, got: %q", out)
		}
		if !strings.Contains(out, "Open Missing Member") {
			t.Fatalf("expected open missing goal in triage output, got: %q", out)
		}
		if strings.Contains(out, "Completed Missing Member") {
			t.Fatalf("did not expect completed missing goal in output, got: %q", out)
		}
	})

	t.Run("preset triage hides completed goals", func(t *testing.T) {
		cmd := triageGoals(service, cfg, writer)
		cmd.SetArgs([]string{"--projectId", project.ID, "--preset", "do-it-now", "--stdout"})

		out := captureStdout(t, func() {
			if err := cmd.ExecuteContext(context.Background()); err != nil {
				t.Fatalf("triage preset command failed: %v", err)
			}
		})

		if !strings.Contains(out, "Open Do It Now") {
			t.Fatalf("expected open goal in preset triage output, got: %q", out)
		}
		if strings.Contains(out, "Completed Do It Now") {
			t.Fatalf("did not expect completed goal in preset triage output, got: %q", out)
		}
	})
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stdout pipe: %v", err)
	}
	os.Stdout = w
	defer func() {
		os.Stdout = oldStdout
	}()

	outCh := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		outCh <- buf.String()
	}()

	fn()

	_ = w.Close()
	out := <-outCh
	_ = r.Close()

	return stripANSI(out)
}
