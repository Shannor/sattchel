# Sattchel

> Named because of my initials and because this CLI Tool is a mix of random tools.

CLI testing with patterns and best practices and random tools that I found useful at times.
It's just for learning and fun.

I'm mainly trying to learn the process of setting up a CLI testing framework with patterns and best practices.
I don't want to use AI too much during this process. Though I'm sure I'll use it here and there for certain tasks when I'm lazy.

## Why? 

- Why Go?
  - Mostly because I already know it. It's good for CLI but, that's why I choose it. It's really because I want to learn other skills and I don't want to fight the language while doing it.
- Why CLI? 
    - Just something I haven't done before and introduces a different challenge.
    - If the Service layer is done well, I should be able to still convert it to a Web Page too by separating the Service layer from the CLI layer.

## Goals

My goals can be followed along [ here if you care ](./LEARNINGS.md) :)

## Installation


### Build from Source (preferred if using Go)

```bash
go install github.com/Shannor/sattchel@latest
sattchel --help
```

That installs the main binary as `sattchel`. The first run automatically creates the `sat` and `satt` aliases in the same Go bin directory.

### From GitHub Releases

Download pre-built binaries directly from GitHub releases:

#### Linux (amd64)

```bash
mkdir -p ~/bin
curl -sSfL https://github.com/Shannor/sattchel/releases/latest/download/sattchel_Linux_x86_64.tar.gz | tar xz -C ~/bin/
```

#### Linux (arm64)

```bash
mkdir -p ~/bin
curl -sSfL https://github.com/Shannor/sattchel/releases/latest/download/sattchel_Linux_arm64.tar.gz | tar xz -C ~/bin/
```

#### macOS (Intel)

```bash
mkdir -p ~/bin
curl -sSfL https://github.com/Shannor/sattchel/releases/latest/download/sattchel_Darwin_x86_64.tar.gz | tar xz -C ~/bin/
```

#### macOS (Apple Silicon)

```bash
mkdir -p ~/bin
curl -sSfL https://github.com/Shannor/sattchel/releases/latest/download/sattchel_Darwin_arm64.tar.gz | tar xz -C ~/bin/
```

#### Windows (amd64)

```powershell
$env:USERPROFILE = $env:USERPROFILE -replace '\\', '/'
Invoke-WebRequest -Uri https://github.com/Shannor/sattchel/releases/latest/download/sattchel_Windows_x86_64.zip -OutFile sattchel.zip
Expand-Archive sattchel.zip -DestinationPath $env:USERPROFILE/temp
copy-item $env:USERPROFILE/temp/sattchel_Windows_x86_64/*.exe $env:USERPROFILE/bin/
Remove-Item sattchel.zip, $env:USERPROFILE/temp -Recurse -Force
```

#### Windows (arm64)

```powershell
$env:USERPROFILE = $env:USERPROFILE -replace '\\', '/'
Invoke-WebRequest -Uri https://github.com/Shannor/sattchel/releases/latest/download/sattchel_Windows_arm64.zip -OutFile sattchel.zip
Expand-Archive sattchel.zip -DestinationPath $env:USERPROFILE/temp
copy-item $env:USERPROFILE/temp/sattchel_Windows_arm64/*.exe $env:USERPROFILE/bin/
Remove-Item sattchel.zip, $env:USERPROFILE/temp -Recurse -Force
```

Or download manually from the [Releases page](https://github.com/Shannor/sattchel/releases) and place the binaries in your PATH.

**Important:** Make sure the install directory is in your PATH.

- Release installs above extract into `~/bin`
- `go install` uses `$GOBIN` when set, otherwise `$(go env GOPATH)/bin`

For release installs, add the following to your shell configuration file (e.g., `~/.bashrc`, `~/.zshrc`, or `~/.profile`):

```bash
export PATH="$HOME/bin:$PATH"
```

Then reload your shell configuration:

```bash
source ~/.bashrc  # or source ~/.zshrc, or source ~/.profile
```

**Verify your PATH:**

Check if `~/bin` is in your PATH:

```bash
echo $PATH | grep "$HOME/bin"
```

If you don't see `~/bin` in the output, your PATH isn't set up correctly. You can also check your current PATH:

```bash
echo $PATH
```

For `go install`, verify the Go bin directory instead:

```bash
bin_dir="${GOBIN:-$(go env GOPATH)/bin}"
echo "$PATH" | grep "$bin_dir"
```

---

## Setup


### Updating

The CLI automatically checks for updates on startup. If a new version is available, you'll see a notification in the terminal.

To manually check for updates:
```bash
sattchel update
```

To force update to the latest version:
```bash
sattchel update --force
```
