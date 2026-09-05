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
4. **Use `cobra.MaximumNArgs` instead of `cobra.ExactArgs`:** Commands intended to support interactive fallback MUST NOT use `cobra.ExactArgs(...)`. Using `ExactArgs` causes Cobra to fail prior to `RunE`, preventing interactive TUI fallback when positional arguments are omitted.
5. **Non-Interactive Validation First:** In `RunE`, always check `if !loader.IsTerminal()` when required positional arguments or flags are omitted and return a clear error message (e.g. `"goal ID is required in non-interactive mode"`).
6. **Explicit Flag Intent in Automation:** When automation/AI passes explicit flags for destructive or recursive actions (e.g. `goals delete <id> -r`), treat the flag as explicit confirmation in non-interactive mode (`if !loader.IsTerminal() { return true, nil }`) rather than failing due to missing TUI prompts.
7. **Bypass Pagers in Automation:** Commands using TUI pagers (`tui.RunPager`) must bypass the pager and write directly to stdout when `!loader.IsTerminal()` or when `--stdout` is set.

---

## Pattern Examples & Learnings

### 1. Update Commands (Flag Validation vs Interactive Form)

```go
func updateMember(service *core.Service, cfg *Config, writer printer.Writer) *cobra.Command {
 var name, email string

 return &cobra.Command{
  Use:          "update [id]",
  Args:         cobra.MaximumNArgs(1),
  SilenceUsage: true,
  RunE: func(cmd *cobra.Command, args []string) error {
   id := ""
   if len(args) > 0 {
    id = args[0]
   }

   // Interactive selection if ID omitted in TTY
   if id == "" {
    if !loader.IsTerminal() {
     return errors.New("member ID is required in non-interactive mode")
    }
    members, err := service.GetMembers(cmd.Context())
    if err != nil {
     return err
    }
    id, err = tui.ChooseMember(members, "Select Member to Update", false)
    if err != nil {
     return err
    }
   }

   hasFlags := cmd.Flags().Changed("name") || cmd.Flags().Changed("email")

   // Non-interactive mode requires at least one flag
   if !hasFlags && !loader.IsTerminal() {
    return errors.New("at least one of --name or --email must be specified for update in non-interactive mode")
   }

   member, err := service.GetMember(cmd.Context(), id)
   if err != nil {
    return err
   }

   // Interactive form fallback pre-filled with current entity values
   if !hasFlags {
    name = member.Name
    email = member.Email
    err = tui.NewForm(
     huh.NewGroup(
      huh.NewInput().Title("Member Name").Value(&name),
      huh.NewInput().Title("Email").Value(&email),
     ),
    ).Run()
    if err != nil {
     return err
    }
    member.Name = name
    member.Email = email
   } else {
    if cmd.Flags().Changed("name") { member.Name = name }
    if cmd.Flags().Changed("email") { member.Email = email }
   }

   updated, err := service.UpdateMember(cmd.Context(), member)
   if err != nil {
    return err
   }
   writer.Success(fmt.Sprintf("Member %s updated successfully", updated.Name))
   return nil
  },
 }
}
```

### 2. Context & Identifier Resolution Helper

```go
func ensureProjectID(cmd *cobra.Command, service *core.Service, cfg *Config, projectIDFlag string) (string, error) {
 pid := getActiveProjectID(cmd, cfg, projectIDFlag)
 if pid != "" {
  return pid, nil
 }
 if !loader.IsTerminal() {
  return "", errors.New("no active project configured and no --projectId flag provided")
 }
 projects, err := service.GetProjects(cmd.Context())
 if err != nil {
  return "", err
 }
 if len(projects) == 0 {
  return "", errors.New("no projects found")
 }
 return tui.ChooseProject(projects, "Select Project", "")
}
```
