# Methodolgy
gh-subissue is vibe-coded on main, human reviewed for tags. Tags generate releases. 

My method here is to ask Claude (normally Opus) to develop features with particular style guidelines in Claude.md, such as "if an option that requires an argument is invoked without the argument, defer to a TUI interface". It's my best guidance to make this tool feel like a GitHub CLI plugin that follows their style.

I am comfortable simply pushing this stuff up, but I do go back and review by hand. It might be some time before I do that, so `main` is kept as mostly vibed most of the time, but tag releases have a human touch.

# gh-subissue

A [GitHub CLI](https://cli.github.com/) extension that creates sub-issues in a single command.

Sub-issues are GitHub's way to break down large issues into smaller, trackable pieces. Normally, creating a sub-issue requires multiple steps: create an issue, then link it to a parent. This extension does it all in one command.

## Installation

**From a release (recommended):**
```bash
gh extension install playtechnique/gh-subissue
```

This downloads a precompiled binary for your platform from the latest [release](https://github.com/gwynforthewyn/gh-subissue/releases).

**From source:**
```bash
git clone https://github.com/gwynforthewyn/gh-subissue.git
cd gh-subissue
gh extension install .
```

**Requirements:**
- [GitHub CLI](https://cli.github.com/) 2.0 or later
- A repository with sub-issues enabled (requires GitHub organization or certain plans)

## Quick Start

```bash
# Create a sub-issue interactively (prompts for parent and title)
gh subissue create

# Create with parent and title specified
gh subissue create --parent 42 --title "Implement authentication"

# List sub-issues under a parent
gh subissue list --parent 42

# Add an existing issue to a project
gh subissue edit 43 --project "Roadmap"
```

## Commands

### `create` - Create a new sub-issue

Creates a new issue and links it to a parent issue in one step.

```bash
gh subissue create [flags]
```

**Flags:**
| Flag | Description |
|------|-------------|
| `-p, --parent <number>` | Parent issue number (interactive if omitted) |
| `-t, --title <string>` | Issue title (interactive if omitted) |
| `-b, --body <string>` | Issue body |
| `--body-file <file>` | Read body from file (use `-` for stdin) |
| `-R, --repo <owner/repo>` | Target repository |
| `-a, --assignee <user>` | Assign users (repeatable) |
| `-l, --label <name>` | Add labels (repeatable) |
| `-m, --milestone <number>` | Add to milestone |
| `-P, --project <name>` | Add to project (interactive if empty string) |
| `-w, --web` | Open in browser after creation |

**Examples:**
```bash
# Fully specified
gh subissue create -p 42 -t "Fix login bug" -l bug -a octocat

# Interactive mode
gh subissue create

# With body from file
gh subissue create -p 42 -t "New feature" --body-file spec.md

# Add to a project
gh subissue create -p 42 -t "Task" --project "Roadmap"
```

### `list` - List sub-issues

Shows all sub-issues linked to a parent issue.

```bash
gh subissue list [flags]
```

**Flags:**
| Flag | Description |
|------|-------------|
| `-p, --parent <number>` | Parent issue number (interactive if omitted) |
| `-R, --repo <owner/repo>` | Target repository |
| `--no-header` | Omit table header from output |

**Example:**
```bash
gh subissue list --parent 42
#  NUMBER  TITLE
#  45      Implement backend
#  46      Add frontend tests
```

### `edit` - Modify a sub-issue

Edit an existing sub-issue (currently supports adding to projects).

```bash
gh subissue edit <issue-number> [flags]
```

**Flags:**
| Flag | Description |
|------|-------------|
| `-P, --project <name>` | Add to project (interactive if empty string) |
| `-R, --repo <owner/repo>` | Target repository |

**Example:**
```bash
gh subissue edit 45 --project "Sprint 3"
```

### `repos` - List repository sub-issue status

Shows which repositories have sub-issues enabled.

```bash
gh subissue repos [<owner>] [flags]
```

**Flags:**
| Flag | Description |
|------|-------------|
| `-L, --limit <int>` | Maximum repositories to list (default: 30) |
| `--enabled` | Show only repos where sub-issues work |
| `--disabled` | Show only repos where sub-issues are disabled |
| `--no-header` | Omit table header from output |

**Example:**
```bash
gh subissue repos my-org --enabled
```

## Repository Resolution

Commands automatically detect the repository context:

1. **`--repo` flag** - Explicit repository (always takes precedence)
2. **`GH_REPO` environment variable** - If set
3. **Git remote** - Auto-detected from current directory
4. **Interactive prompt** - If all above fail and running interactively

## Environment Variables

| Variable | Description |
|----------|-------------|
| `GH_REPO` | Override repository resolution |
| `GH_DEBUG` | Enable debug logging (set to any value) |

## Troubleshooting

### "Sub-issues are not enabled for this repository"

Sub-issues require specific GitHub plans or organization settings. Check your repository's settings or run:

```bash
gh subissue repos --enabled
```

### "Could not determine repository"

Run the command inside a git repository with a GitHub remote, or specify the repository explicitly:

```bash
gh subissue create --repo owner/repo --parent 42 --title "Task"
```

## Contributing

Contributions are welcome! Please feel free to submit issues and pull requests.

### Development

```bash
# Clone the repository
git clone https://github.com/gwynforthewyn/gh-subissue.git
cd gh-subissue

# Build
go build -o gh-subissue

# Run tests
go test ./...

# Install locally for testing
gh extension install .
```

### Testing

This project uses strict TDD with Go's standard library `testing` package. Please write tests before implementing features.

## License

MIT License - see [LICENSE](LICENSE) for details.
