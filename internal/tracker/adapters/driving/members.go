package driving

import (
	"fmt"
	"strings"

	"sattchel/internal/printer"
	"sattchel/internal/tracker/core"
	"sattchel/internal/tui"
	"sattchel/pkg/loader"

	"charm.land/huh/v2"
	"github.com/spf13/cobra"
)

func members(service *core.Service, cfg *Config, writer printer.Writer) *cobra.Command {
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
	cmd.AddCommand(createMember(service, cfg, writer))
	cmd.AddCommand(getMember(service, cfg, writer))
	cmd.AddCommand(listMembers(service, cfg, writer))
	cmd.AddCommand(updateMember(service, cfg, writer))
	cmd.AddCommand(deleteMember(service, cfg, writer))
	return cmd
}

func createMember(service *core.Service, cfg *Config, writer printer.Writer) *cobra.Command {
	var email string
	cmd := &cobra.Command{
		Use:          "create [name]",
		Aliases:      []string{"add"},
		Short:        "Create a new member",
		Args:         cobra.MaximumNArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			name := ""
			if len(args) > 0 {
				name = args[0]
			}

			if name == "" {
				if !loader.IsTerminal() {
					return fmt.Errorf("member name is required in non-interactive mode")
				}
				err := tui.NewForm(
					huh.NewGroup(
						huh.NewInput().
							Title("Member Name").
							Value(&name).
							Validate(func(str string) error {
								if strings.TrimSpace(str) == "" {
									return fmt.Errorf("member name is required")
								}
								return nil
							}),
						huh.NewInput().
							Title("Email").
							Value(&email),
					),
				).Run()
				if err != nil {
					return err
				}
			}

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
			writer.Success(fmt.Sprintf("Member %s (%s) created successfully", member.Name, member.ID))
			return nil
		},
	}
	cmd.Flags().StringVarP(&email, "email", "e", "", "email of the member")
	return cmd
}

func getMember(service *core.Service, cfg *Config, writer printer.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "get [id]",
		Aliases:      []string{"view", "show"},
		Short:        "Get a member's details",
		Args:         cobra.MaximumNArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			id := ""
			if len(args) > 0 {
				id = args[0]
			}
			if id == "" {
				if !loader.IsTerminal() {
					return fmt.Errorf("member ID is required in non-interactive mode")
				}
				var members []core.Member
				var err error
				_ = loader.Run("Getting members list...", func() {
					members, err = service.GetMembers(cmd.Context())
				})
				if err != nil {
					return err
				}
				if len(members) == 0 {
					writer.Info("No members found")
					return nil
				}
				selectedID, err := tui.ChooseMember(members, "Select Member to View", false)
				if err != nil {
					return err
				}
				id = selectedID
			}

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
			var emailStr string
			if member.Email != "" {
				emailStr = fmt.Sprintf(" - %s", member.Email)
			}
			writer.Info(fmt.Sprintf("Member: %s (%s)%s", member.Name, member.ID, emailStr))
			return nil
		},
	}
	return cmd
}

func listMembers(service *core.Service, cfg *Config, writer printer.Writer) *cobra.Command {
	var stdoutFlag bool
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
				writer.Info("No members found")
				return nil
			}

			if loader.IsTerminal() && !stdoutFlag {
				selectedID, err := tui.ChooseMember(members, "Select Member to View", false)
				if err != nil {
					return err
				}
				for _, m := range members {
					if m.ID == selectedID {
						var emailStr string
						if m.Email != "" {
							emailStr = fmt.Sprintf(" - %s", m.Email)
						}
						writer.Info(fmt.Sprintf("Member: %s (%s)%s", m.Name, m.ID, emailStr))
						break
					}
				}
				return nil
			}

			for _, m := range members {
				var emailStr string
				if m.Email != "" {
					emailStr = fmt.Sprintf(" - %s", m.Email)
				}
				writer.Info(fmt.Sprintf("- %s (%s)%s", m.Name, m.ID, emailStr))
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&stdoutFlag, "stdout", false, "Output plain text instead of interactive selector")
	return cmd
}

func updateMember(service *core.Service, cfg *Config, writer printer.Writer) *cobra.Command {
	var name string
	var email string
	cmd := &cobra.Command{
		Use:          "update [id]",
		Aliases:      []string{"edit"},
		Short:        "Update a member's details",
		Args:         cobra.MaximumNArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			id := ""
			if len(args) > 0 {
				id = args[0]
			}
			if id == "" {
				if !loader.IsTerminal() {
					return fmt.Errorf("member ID is required in non-interactive mode")
				}
				members, err := service.GetMembers(cmd.Context())
				if err != nil {
					return err
				}
				if len(members) == 0 {
					return fmt.Errorf("no members found")
				}
				selectedID, err := tui.ChooseMember(members, "Select Member to Update", false)
				if err != nil {
					return err
				}
				id = selectedID
			}

			if !cmd.Flags().Changed("name") && !cmd.Flags().Changed("email") {
				if !loader.IsTerminal() {
					return fmt.Errorf("at least one of --name or --email must be specified for update in non-interactive mode")
				}
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

			if !cmd.Flags().Changed("name") && !cmd.Flags().Changed("email") {
				name = member.Name
				email = member.Email
				err = tui.NewForm(
					huh.NewGroup(
						huh.NewInput().
							Title("Member Name").
							Value(&name).
							Validate(func(str string) error {
								if strings.TrimSpace(str) == "" {
									return fmt.Errorf("member name is required")
								}
								return nil
							}),
						huh.NewInput().
							Title("Email").
							Value(&email),
					),
				).Run()
				if err != nil {
					return err
				}
				member.Name = name
				member.Email = email
			} else {
				if cmd.Flags().Changed("name") {
					member.Name = name
				}
				if cmd.Flags().Changed("email") {
					member.Email = email
				}
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
			writer.Success(fmt.Sprintf("Member %s (%s) updated successfully", member.Name, member.ID))
			return nil
		},
	}
	cmd.Flags().StringVarP(&name, "name", "n", "", "New name of the member")
	cmd.Flags().StringVarP(&email, "email", "e", "", "New email address of the member")
	return cmd
}

func deleteMember(service *core.Service, cfg *Config, writer printer.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "delete [id]",
		Aliases:      []string{"remove", "rm"},
		Short:        "Delete a member",
		Args:         cobra.MaximumNArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			id := ""
			if len(args) > 0 {
				id = args[0]
			}
			if id == "" {
				if !loader.IsTerminal() {
					return fmt.Errorf("member ID is required in non-interactive mode")
				}
				members, err := service.GetMembers(cmd.Context())
				if err != nil {
					return err
				}
				if len(members) == 0 {
					return fmt.Errorf("no members found")
				}
				selectedID, err := tui.ChooseMember(members, "Select Member to Delete", false)
				if err != nil {
					return err
				}
				id = selectedID
			}

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
			writer.Success(fmt.Sprintf("Member %s deleted successfully", id))
			return nil
		},
	}
	return cmd
}
