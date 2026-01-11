# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

gh-subissue is a GitHub CLI extension that creates sub-issues in a single command. It wraps the multi-step process of creating an issue and linking it to a parent into one `gh subissue create` command.

## Build & Development Commands

```bash
# Build the extension
go build -o gh-subissue

# Run tests
go test ./...

# Run a single test
go test -run TestName ./path/to/package

# Install locally for testing
gh extension install .

# Uninstall
gh extension remove subissue
```

## Architecture

### API Flow
1. Parse flags and resolve repository context (via go-gh)
2. Create issue via `POST /repos/{owner}/{repo}/issues`
3. Extract the returned issue's `id` field (internal numeric ID, not issue number)
4. Link to parent via `POST /repos/{owner}/{repo}/issues/{parent}/sub_issues` with `{"sub_issue_id": id}`

### Key Dependencies
- `github.com/cli/go-gh/v2` — core extension SDK, handles auth and repo resolution
- `github.com/cli/go-gh/v2/pkg/api` — REST/GraphQL client
- `github.com/spf13/cobra` — CLI framework

### Repository Resolution Order
1. Explicit `--repo` flag
2. `GH_REPO` environment variable
3. Git remote in current directory

### Error Handling Pattern
If issue creation succeeds but sub-issue linking fails, print a warning with the issue URL and manual linking instructions rather than failing silently.

## Testing Approach

This project uses **strict TDD** (Test-Driven Development):

1. **Write tests first** — before implementing any feature or fix
2. **Use Go's stdlib `testing` package** — no external testing frameworks (no testify, gomega, etc.)
3. **Red-Green-Refactor cycle** — write a failing test, make it pass, then refactor

## Git Commit Practices

1. **Separate commits for each feature** — each logical change gets its own commit
2. **Interface changes first** — when adding methods to interfaces, commit the interface change separately from the feature that uses it
3. **Atomic commits** — each commit should pass tests independently

## Interface & Mock Maintenance

When modifying interfaces (e.g., `Prompter` in `cmd/prompter.go`):

1. **Update all mock implementations** — search for structs implementing the interface and add the new method
2. **Check test files** — mocks often live in `*_test.go` files (e.g., `mockPrompter` in `select_test.go`, `mockPrompterInCreate` in `create_test.go`)
3. **Use compile-time checks** — ensure `var _ Interface = (*mockType)(nil)` assertions exist to catch missing methods
