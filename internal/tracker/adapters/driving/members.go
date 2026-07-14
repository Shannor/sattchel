package driving

import (
	"fmt"
	"sattchel/internal/tracker/core"
	"sattchel/internal/tui"
	"sattchel/pkg/loader"

	"github.com/spf13/cobra"
)

func members(service *core.Service, cfg *Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "member",
		Aliases: []string{"m", "members"},
		Short:   "Manage members",
		Long: `Manage members.
   Examples:
     satt tracker member create <name> --email <email>
     satt tracker member get <id>
     satt tracker member list
     satt tracker member update <id> --name <new_name> --email <new_email>
     satt tracker member delete <id>
`,
	}
	cmd.AddCommand(createMember(service, cfg))
	cmd.AddCommand(getMember(service, cfg))
	cmd.AddCommand(listMembers(service, cfg))
	cmd.AddCommand(updateMember(service, cfg))
	cmd.AddCommand(deleteMember(service, cfg))
	return cmd
}

func createMember(service *core.Service, cfg *Config) *cobra.Command {
	var email string
	cmd := &cobra.Command{
		Use:          "create <name>",
		Aliases:      []string{"add"},
		Short:        "Create a new member",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			var member *core.Member
			var err error
			runErr := loader.Run("Creating member...", func() {
				member, err = service.CreateMember(cmd.Context(), name, email)
			})
			if runErr != nil {
				return runErr
			}
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Member created successfully:\nID: %s\nName: %s\nEmail: %s\n", member.ID, member.Name, member.Email)
			return nil
		},
	}
	cmd.Flags().StringVarP(&email, "email", "e", "", "email of the member")
	return cmd
}

func getMember(service *core.Service, cfg *Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "get <id>",
		Aliases:      []string{"view", "show"},
		Short:        "Get a member's details",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			var member *core.Member
			var err error
			runErr := loader.Run("Getting member details...", func() {
				member, err = service.GetMember(cmd.Context(), id)
			})
			if runErr != nil {
				return runErr
			}
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "ID: %s\nName: %s\nEmail: %s\n", member.ID, member.Name, member.Email)
			return nil
		},
	}
	return cmd
}

func listMembers(service *core.Service, cfg *Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "list",
		Short:        "List all members",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			var members []core.Member
			var err error
			runErr := loader.Run("Getting members list...", func() {
				members, err = service.GetMembers(cmd.Context())
			})
			if runErr != nil {
				return runErr
			}
			if err != nil {
				return err
			}
			if len(members) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No members found")
				return nil
			}

			if loader.IsTerminal() {
				var options []tui.ListOption
				for _, m := range members {
					desc := m.ID
					if m.Email != "" {
						desc = fmt.Sprintf("%s — %s", m.Email, m.ID)
					}
					options = append(options, tui.ListOption{
						TitleStr:       m.Name,
						DescriptionStr: desc,
						ValueStr:       m.ID,
					})
				}
				selected, err := tui.Choose("Select Member to View", options)
				if err != nil {
					return err
				}
				if selected != nil {
					for _, m := range members {
						if m.ID == selected.ValueStr {
							if m.Email != "" {
								fmt.Fprintf(cmd.OutOrStdout(), "ID: %s\nName: %s\nEmail: %s\n", m.ID, m.Name, m.Email)
							} else {
								fmt.Fprintf(cmd.OutOrStdout(), "ID: %s\nName: %s\n", m.ID, m.Name)
							}
							break
						}
					}
				}
				return nil
			}

			for _, m := range members {
				if m.Email != "" {
					fmt.Fprintf(cmd.OutOrStdout(), "- %s (%s) - %s\n", m.Name, m.ID, m.Email)
				} else {
					fmt.Fprintf(cmd.OutOrStdout(), "- %s (%s)\n", m.Name, m.ID)
				}
			}
			return nil
		},
	}
	return cmd
}

func updateMember(service *core.Service, cfg *Config) *cobra.Command {
	var name string
	var email string
	cmd := &cobra.Command{
		Use:          "update <id>",
		Aliases:      []string{"edit"},
		Short:        "Update a member's details",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]

			if !cmd.Flags().Changed("name") && !cmd.Flags().Changed("email") {
				return fmt.Errorf("at least one of --name or --email must be specified for update")
			}

			var member *core.Member
			var err error
			runErr := loader.Run("Fetching member...", func() {
				member, err = service.GetMember(cmd.Context(), id)
			})
			if runErr != nil {
				return runErr
			}
			if err != nil {
				return err
			}

			if cmd.Flags().Changed("name") {
				member.Name = name
			}
			if cmd.Flags().Changed("email") {
				member.Email = email
			}

			runErr = loader.Run("Updating member...", func() {
				member, err = service.UpdateMember(cmd.Context(), member)
			})
			if runErr != nil {
				return runErr
			}
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Member updated successfully:\nID: %s\nName: %s\nEmail: %s\n", member.ID, member.Name, member.Email)
			return nil
		},
	}
	cmd.Flags().StringVarP(&name, "name", "n", "", "New name of the member")
	cmd.Flags().StringVarP(&email, "email", "e", "", "New email address of the member")
	return cmd
}

func deleteMember(service *core.Service, cfg *Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "delete <id>",
		Aliases:      []string{"remove", "rm"},
		Short:        "Delete a member",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			var err error
			runErr := loader.Run("Deleting member...", func() {
				err = service.DeleteMember(cmd.Context(), id)
			})
			if runErr != nil {
				return runErr
			}
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Member %s deleted successfully\n", id)
			return nil
		},
	}
	return cmd
}
