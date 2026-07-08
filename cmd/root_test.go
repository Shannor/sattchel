package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestIsCompletionCommand(t *testing.T) {
	t.Parallel()

	root := &cobra.Command{Use: "sattchel"}
	completion := &cobra.Command{Use: "completion"}
	zsh := &cobra.Command{Use: "zsh"}
	tracker := &cobra.Command{Use: "tracker"}
	complete := &cobra.Command{Use: "__complete"}
	completeNoDesc := &cobra.Command{Use: "__completeNoDesc"}

	root.AddCommand(completion, tracker, complete, completeNoDesc)
	completion.AddCommand(zsh)

	tests := []struct {
		name string
		cmd  *cobra.Command
		want bool
	}{
		{name: "regular command", cmd: tracker, want: false},
		{name: "completion command", cmd: completion, want: true},
		{name: "completion subcommand", cmd: zsh, want: true},
		{name: "hidden complete command", cmd: complete, want: true},
		{name: "hidden complete no desc command", cmd: completeNoDesc, want: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := isCompletionCommand(tt.cmd); got != tt.want {
				t.Fatalf("isCompletionCommand() = %v, want %v", got, tt.want)
			}
		})
	}
}
