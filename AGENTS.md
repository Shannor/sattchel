# Sattchel

CLI tool with `satt` as the command.

## Supports

- Optimizely (feature flags)
  - compare
  - unique
  - dormant
  - drift
  - promote
  - sync
- Tracker (mindmap and task tracking)

## Code Structure Rules

- **Driving Adapters (Action-Per-File)**: Split driving files into one file per action once they grow long.
  - `<resource>.go` registers subcommands.
  - `<resource>_<action>.go` defines the individual action (e.g. `goals_add.go`, `goals_delete.go`, `projects_create.go`).

## Git & Release Rules

- **Never Overwrite Tags**: Never push to or re-point an existing Git tag. Always create a new version tag for new changes or releases.
