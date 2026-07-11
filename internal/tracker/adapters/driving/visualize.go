package driving

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"sattchel/internal/tracker/core"

	"github.com/spf13/cobra"
)

func visualizeProject(service *core.Service, cfg *Config) *cobra.Command {
	projectID := ""

	cmd := &cobra.Command{
		Use:   "visualize",
		Short: "Start the visualizer web server for a project",
		Long: `Start an ephemeral local web server to visualize a project's goals as an interactive mind map.
Automatically opens the mind map in your default browser.
Examples:
  satt tracker visualize
  `,
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			pid := projectID
			if !cmd.Flags().Changed("projectId") {
				if lastProj := cfg.CurrentProjectID(); lastProj != "" {
					pid = lastProj
				}
			}
			if pid == "" {
				return fmt.Errorf("no project selected")
			}

			fmt.Println("Getting goals ...")
			goals, err := service.GetGoals(cmd.Context(), pid)
			if err != nil {
				return err
			}
			if len(goals) == 0 {
				return fmt.Errorf("no goals found for project %s", pid)
			}

			fmt.Println("Starting visualizer server ...")
			server := NewHTTPServer(service)
			addr, shutdown, err := server.Start(cmd.Context(), "127.0.0.1:0")
			if err != nil {
				return fmt.Errorf("failed to start server: %w", err)
			}

			url := fmt.Sprintf("http://%s?projectId=%s", addr, pid)

			fmt.Printf("Visualizer server running at: %s\n", url)
			fmt.Println("Opening in browser...")
			_ = openBrowser(url)

			fmt.Println("Press Ctrl+C to stop the visualizer server.")

			// Wait for interrupt signal to stop the server
			stop := make(chan os.Signal, 1)
			signal.Notify(stop, os.Interrupt)
			<-stop

			fmt.Println("\nStopping server ...")
			if err := shutdown(); err != nil {
				fmt.Printf("Error stopping server: %v\n", err)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&projectID, "projectId", "p", "", "Project id of the goals. If not provided, the default project will be used")
	_ = cmd.RegisterFlagCompletionFunc("projectId", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return getProjectCompletions(service), cobra.ShellCompDirectiveNoFileComp
	})
	return cmd
}

func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default: // "linux", "freebsd", etc.
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Run()
}
