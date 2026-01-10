# gh-subissue

A GitHub CLI extension to create sub-issues in a single command.

## Problem

GitHub's sub-issues feature allows parent/child relationships between issues, but the `gh` CLI doesn't support creating an issue with a parent in one step. Currently you must:

1. Create the issue with `gh issue create`
2. Retrieve the issue's internal ID via `gh api`
3. Link it to a parent via `gh api repos/.../issues/PARENT/sub_issues`

This plugin combines these steps into a single command.

## Installation

```bash
gh extension install <owner>/gh-subissue
```

## Usage

```bash
# Create a sub-issue under parent issue #42
gh subissue create --parent 42 --title "Implement feature X" --body "Details here"

# Create a sub-issue in a specific repo
gh subissue create --repo owner/repo --parent 42 --title "Fix bug"

# Interactive mode (like gh issue create)
gh subissue create --parent 42

# Pipe body from stdin
echo "Task details" | gh subissue create --parent 42 --title "New task" --body-file -
```

## Command Reference

### `gh subissue create`

Creates a new issue and immediately links it as a sub-issue to a parent.

| Flag | Description |
|------|-------------|
| `--parent`, `-p` | **Required.** Parent issue number |
| `--title`, `-t` | Issue title (prompts if omitted) |
| `--body`, `-b` | Issue body |
| `--body-file` | Read body from file (use `-` for stdin) |
| `--repo`, `-R` | Repository in `owner/repo` format (defaults to current repo) |
| `--assignee`, `-a` | Assign users (can be repeated) |
| `--label`, `-l` | Add labels (can be repeated) |
| `--milestone`, `-m` | Add to milestone |
| `--web`, `-w` | Open the created issue in browser |

## Implementation Notes

### Language

Go — required for `gh` extensions that want to be distributed as precompiled binaries via `gh extension install`.

### Dependencies

Use the `gh` extension libraries:

- `github.com/cli/go-gh/v2` — core extension SDK
- `github.com/cli/go-gh/v2/pkg/api` — REST/GraphQL client with auth handled

### API Flow

1. Parse flags and resolve repository context
2. Create the issue via `POST /repos/{owner}/{repo}/issues`
3. Extract the returned issue's `id` field (internal numeric ID, not issue number)
4. Link to parent via `POST /repos/{owner}/{repo}/issues/{parent}/sub_issues` with `{"sub_issue_id": id}`
5. Print the created issue URL

### Error Handling

- If issue creation succeeds but sub-issue linking fails, print a warning with the issue URL and instructions to manually link it — don't leave the user with a silent failure
- Validate that parent issue exists before creating the child (optional, but better UX)
- Handle rate limiting gracefully

### Repository Resolution

Use `go-gh` repo resolution which checks:

1. Explicit `--repo` flag
2. `GH_REPO` environment variable  
3. Git remote in current directory

### Authentication

Handled automatically by `go-gh` — uses the same auth as the `gh` CLI.

## Project Structure

```
gh-subissue/
├── main.go           # Entry point, root command setup
├── cmd/
│   └── create.go     # create subcommand implementation
├── internal/
│   └── api/
│       └── issues.go # API calls for issue creation and linking
├── go.mod
├── go.sum
└── README.md
```

## Testing

- Unit tests for flag parsing and API request construction
- Integration tests against a real repo (use `GH_TOKEN` in CI)
- Consider using `go-gh`'s mock HTTP client for unit tests

## Release

Use GoReleaser with the `gh extension` preset. The `.goreleaser.yml` should build binaries for:

- `darwin-amd64`
- `darwin-arm64`  
- `linux-amd64`
- `linux-arm64`
- `windows-amd64`

This enables `gh extension install` to download precompiled binaries instead of requiring Go on the user's machine.

## Future Enhancements

- `gh subissue list` — list sub-issues of a parent
- `gh subissue add` — link an existing issue as a sub-issue
- `gh subissue remove` — unlink a sub-issue from its parent
- Support for issue templates
- `--type` flag for issue types (once that API stabilizes)