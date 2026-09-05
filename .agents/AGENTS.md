# Custom Agent Rules

- Use the Makefile for building the application.
- When working locally, the CLI artifact will be `./bin/satt` and the local config will be in `dev-config/`
- Do not give error variables custom names (e.g. runErr, getGoalsErr). They must always be named `err`.

### CLI Format: The "Modern Cloud / Noun-Verb" Style (Structured Subcommands)

Examples: 
- docker container create
- gh issue list
- aws ec2 start-instances 
- kubectl get pods

Important Notes:
- How it works: Highly structured tree hierarchy. First command is the resource (Noun), second command is the action (Verb):  tool <noun> <verb> [flags] .
- All commands should support a interactive mode for a user and non-interactive for scripting and automation uses cases.