# Test CLI

CLI testing with patterns and best practices.
It's just for learning and fun if I think of a use case. 

I'm mainly trying to learn the process of setting up a CLI testing framework with patterns and best practices.
I don't want to use AI too much during this process. Though I'm sure I'll use it here and there for certain tasks when I'm lazy.

## Why? 

- Why Go?
  - Mostly because I already know it. It's good for CLI but, that's why I choose it. It's really because I want to learn other skills and I don't want to fight the language while doing it.
- Why CLI? 
    - Just something I haven't done before and introduces a different challenge.
    - If the Service layer is done well, I should be able to still convert it to a Web Page too by separating the Service layer from the CLI layer.

## Goals

- [x] Setup a Simple CLI
- [x] Add GitHub installation instructions
- [] Write tests for CLI commands 
- [] Write tests for CLI commands
- [] Setup an Install and Update Flow
- [] Add commands useful for Optimizely
- [] Use Arch/Business Patterns that I'm learning
  - Document the patterns being used and why so they are getting commit to my memory of why I thought they were good at the time.
- [] More comfortable with Architecture Patterns

## Installation

### From GitHub Releases

Download pre-built binaries directly from GitHub releases:

```bash
# Linux (amd64)
curl -sSfL https://github.com/Shannor/test-cli/releases/latest/download/test-cli-linux-amd64.tar.gz | tar xz -C /usr/local/bin/

# Linux (arm64)
curl -sSfL https://github.com/Shannor/test-cli/releases/latest/download/test-cli-linux-arm64.tar.gz | tar xz -C /usr/local/bin/

# macOS (Intel)
curl -sSfL https://github.com/Shannor/test-cli/releases/latest/download/test-cli-darwin-amd64.tar.gz | tar xz -C /usr/local/bin/

# macOS (Apple Silicon)
curl -sSfL https://github.com/Shannor/test-cli/releases/latest/download/test-cli-darwin-arm64.tar.gz | tar xz -C /usr/local/bin/

# Windows (PowerShell)
Invoke-WebRequest -Uri https://github.com/Shannor/test-cli/releases/latest/download/test-cli-windows-amd64.zip -OutFile test-cli.zip
Expand-Archive test-cli.zip -DestinationPath C:\\temp
Move-Item C:\\temp\\test-cli-windows-amd64\\test-cli.exe C:\\Windows\\System32\\
Remove-Item test-cli.zip, C:\\temp -Recurse -Force
```

Or download manually from the [Releases page](https://github.com/Shannor/test-cli/releases) and place the binary in your PATH.

### Build from Source

```bash
go install github.com/Shannor/test-cli/cmd/test-cli@latest
```

---

## Setup


### Updating

The CLI automatically checks for updates on startup. If a new version is available, you'll see a notification in the terminal.

To manually check for updates:
```bash
test-cli update
```

To force update to the latest version:
```bash
test-cli update --force
```

---

## Patterns

The main Architecture Patterns used so far in this project are:
- Service Layer
- Repository Pattern
- Module Pattern 
- Dependency Injection

I'm trying to use a Domain Model Pattern without a fully Object-Oriented Programming language. Therefore, some
patterns will mostly likely change based on how Golang works and it's features and limitations. 
Some patterns won't match at all to their textbook examples. But that's another fun challenge for me to understand
what patterns can work or can be modified to fit my needs.