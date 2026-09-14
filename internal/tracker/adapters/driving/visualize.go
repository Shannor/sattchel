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
	port := 8765
	noOpen := false

	cmd := &cobra.Command{
		Use:   "visualize",
		Short: "Start the visualizer web server for a project",
		Long: `Start an ephemeral local web server to visualize a project's goals as an interactive mind map.
Automatically opens the mind map in your default browser.
Examples:
  satt tracker visualize
  satt tracker visualize --port 8765
  satt tracker visualize --no-open
  `,
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			pid, err := ensureProjectID(cmd, service, cfg, projectID)
			if err != nil {
				return err
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
			listenAddr := fmt.Sprintf("127.0.0.1:%d", port)
			addr, shutdown, err := server.Start(cmd.Context(), listenAddr)
			if err != nil && port != 0 {
				// Fallback to ephemeral port if requested port is unavailable
				addr, shutdown, err = server.Start(cmd.Context(), "127.0.0.1:0")
			}
			if err != nil {
				return fmt.Errorf("failed to start server: %w", err)
			}

			url := fmt.Sprintf("http://%s?projectId=%s", addr, pid)

			fmt.Printf("Visualizer server running at: %s\n", url)
			if !noOpen {
				fmt.Println("Opening in browser...")
				_ = openBrowser(url)
			}

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
	cmd.Flags().IntVarP(&port, "port", "P", 8765, "Port for the visualizer web server")
	cmd.Flags().BoolVar(&noOpen, "no-open", false, "Do not automatically open browser")
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
