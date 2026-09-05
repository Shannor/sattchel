# Repository Skills

Custom repository-specific skills for agents can be added to subdirectories within this folder.

## Directory Structure

Each custom skill should reside in its own subdirectory containing a `SKILL.md` file:

```text
.agents/skills/
└── <skill-name>/
    └── SKILL.md
```

## Skill File Format

```markdown
---
name: skill-name
description: A clear description of what this skill provides and when to use it.
---

# Skill Name Instructions

Detailed instructions, commands, or workflows for the agent to follow when executing this skill.
```
