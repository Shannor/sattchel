package driving

import (
	"fmt"
	"sattchel/internal/tracker/core"

	"github.com/spf13/cobra"
)

func NewCommand(service *core.Service) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "tracker",
		Short:   "Tracker Commands",
		Aliases: []string{"tr"},
	}
	cmd.AddCommand(projects(service))
	cmd.AddCommand(goals(service))
	return cmd
}
func projects(service *core.Service) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "project [next]",
		Aliases: []string{"p"},
		Short:   "Manage projects",
		Long: `Manage projects.
   Examples:
     sattchel tracker project create <name> 
     sattchel tracker project list"
     `,
	}
	cmd.AddCommand(createProject(service))
	return cmd
}

func createProject(service *core.Service) *cobra.Command {
	description := ""
	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new project",
		Long: `Create a new project.
	If no key is provided, a list of available keys will be displayed.
   Examples:
     sattchel tracker project create <name> 
     sattchel tracker project create <name> -d "description"
     `,
		Args:         cobra.MaximumNArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("project name is required")
			}
			p, err := service.CreateProject(cmd.Context(), args[0], description)
			if err != nil {
				return err
			}
			fmt.Printf("Project %s created successfully\n", p.Label)
			return nil
		},
	}
	cmd.Flags().StringVarP(&description, "description", "d", "", "description of the project")
	return cmd
}

func goals(service *core.Service) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "goals [next]",
		Short:   "Manage goals",
		Aliases: []string{"g"},
		Long: `Manage goals.
   Examples:
     sattchel tracker goals create <name>
     sattchel tracker goals list"
     `,
	}
	cmd.AddCommand(createGoal(service))
	return cmd
}

func createGoal(service *core.Service) *cobra.Command {
	description := ""
	parentID := ""
	projectID := ""
	cmd := &cobra.Command{
		Use:   "add <name>",
		Short: "Add a new goal",
		Long: `Add a new goal.
	If no key is provided, a list of available keys will be displayed.
   Examples:
     sattchel tracker goal add <name> 
     sattchel tracker goal add <name> -d "description"
     `,
		Args:         cobra.MaximumNArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("goal name is required")
			}
			options := core.GoalOptions{
				ParentID:    parentID,
				Description: description,
			}
			goal, err := service.CreateGoal(cmd.Context(), projectID, args[0], options)
			if err != nil {
				return err
			}
			fmt.Printf("Goal %s created successfully\n", goal.Name)
			return nil
		},
	}
	cmd.Flags().StringVarP(&description, "description", "d", "", "Description of the goal")
	cmd.Flags().StringVarP(&projectID, "projectId", "p", "7d47a039-bbc8-4799-95c3-075cacd92168", "Project id of the goal. If not provided, the default project will be used")
	return cmd
}
