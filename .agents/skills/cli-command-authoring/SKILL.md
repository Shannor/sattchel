---
name: cli-command-authoring
description: Guide for creating Sattchel CLI subcommands that support both human interactive form flows (huh TUI) and AI/script non-interactive flag flows.
---

# CLI Command Authoring Guidelines

Sattchel CLI (`satt`) commands are driving adapters located in `internal/<component>/adapters/driving/`. Commands must be designed for both **human terminal users** and **AI/automation scripts**.

---

## Command Hierarchy: Noun-Verb Pattern

Commands follow the **Modern Cloud / Noun-Verb** tree structure:
`satt <noun> <verb> [flags] [args]`

Examples:

- `satt tracker goals add [name]`
- `satt tracker goals list`
- `satt optimizely flags compare`

---

## Dual Mode Execution Flow

Every mutation or data-entry command MUST support two execution modes:

1. **Non-Interactive Mode (AI Agents & Automation):**
   - Triggered when positional arguments or explicit flags are supplied, or when stdout/stdin is not a TTY.
   - Executes immediately without prompting or blocking for input.

2. **Interactive Mode (Human Terminal Users):**
   - Triggered when required positional arguments are omitted in an interactive terminal (`loader.IsTerminal()`).
   - Presents interactive form flows using `charm.land/huh` or TUI helpers (`tui.NewForm`).

### Example Implementation Pattern

```go
func addGoal(service *core.Service, cfg *Config, writer printer.Writer) *cobra.Command {
 var description, parentID string

 cmd := &cobra.Command{
  Use:          "add [name]",
  Short:        "Add a new goal",
  Args:         cobra.MaximumNArgs(1),
  SilenceUsage: true,
  RunE: func(cmd *cobra.Context, args []string) error {
   // Non-Interactive flow: Argument supplied directly
   if len(args) > 0 {
    goal, err := service.CreateGoal(cmd.Context(), args[0], description)
    if err != nil {
     return err
    }
    writer.Success(fmt.Sprintf("Goal %s created", goal.Name))
    return nil
   }

   // Interactive flow: No argument supplied in terminal
   if !loader.IsTerminal() {
    return errors.New("goal name is required in non-interactive mode")
   }
   return addGoalInteractive(cmd.Context(), service, cfg)
  },
 }

 cmd.Flags().StringVarP(&description, "description", "d", "", "Description of the goal")
 return cmd
}
```

---

## Interactive Form Guidelines

- Use `charm.land/huh` forms for terminal interactions.
- Wrap asynchronous calls or data-loading steps in `loader.Run(...)`:

  ```go
  var goals []core.Goal
  var err error
  _ = loader.Run("Loading goals...", func() {
      goals, err = service.GetGoals(ctx, projectID)
  })
  if err != nil {
      return err
  }
  ```

- Use `tui.NewForm(...)` to run `huh` groups cleanly.

---

## Shell Completions & Help Text

- Provide clear examples in `cobra.Command.Long` help text showing both non-interactive and interactive usage.
- Register completion handlers for dynamic flags using `RegisterFlagCompletionFunc`:

  ```go
  _ = cmd.RegisterFlagCompletionFunc("parent", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
      return getGoalCompletions(service), cobra.ShellCompDirectiveNoFileComp
  })
  ```

---

## Mandatory Code Rules

1. **Error Variable Naming:** Always name error variables `err`. Custom names like `cmdErr` or `addErr` are strictly prohibited.
2. **Silence Usage on Errors:** Set `SilenceUsage: true` on commands so runtime errors do not print full command help text.
3. **Output Formatting:** Use `printer.Writer` for status messages (`writer.Success(...)`, `writer.Info(...)`).
