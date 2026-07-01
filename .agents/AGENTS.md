# Custom Agent Rules

- Always activate Planning Mode for every request.
- Do not make any code changes, create files, or run terminal commands without first proposing an implementation plan (`implementation_plan.md`) and waiting for explicit user approval.
- Use the Makefile for building the application. 
- When working locally the CLI artifact will be `./bin/sattchel`


### CLI Format: The "Modern Cloud / Noun-Verb" Style (Structured Subcommands)

Examples:  docker container create ,  gh issue list ,  aws ec2 start-instances ,  kubectl get pods

• How it works: Highly structured tree hierarchy. First command is the resource (Noun), second command is the action (Verb):  tool <noun> <verb> [flags] .