# Implementation Plan: Interactive Parent Issue Selection

## Goal

When `--parent` flag is omitted from `gh subissue create`, display an interactive TUI prompt listing candidate parent issues from the repository, allowing the user to select one.

## Research Summary

### TUI Library

The official GitHub CLI (`cli/cli`) uses:

- **Primary**: `github.com/cli/go-gh/v2/pkg/prompter` - wraps `survey/v2` for interactive prompts
- **Fallback**: `github.com/charmbracelet/huh` for accessible mode (screen readers)

Since this project already uses `go-gh/v2`, use `pkg/prompter` directly:

```go
import "github.com/cli/go-gh/v2/pkg/prompter"

p := prompter.New(stdin, stdout, stderr)
idx, err := p.Select("Select parent issue", "", options)
```

### Issue Display Format

GitHub CLI formats issues as: `#123  Issue title   label1, label2   5m ago`

For the selector, a simpler format works: `#123 Issue title` (truncated to ~60 chars)

### API for Listing Issues

Use REST API: `GET /repos/{owner}/{repo}/issues?state=open&per_page=30`

This returns open issues sorted by creation date (newest first). The existing `internal/api` package pattern should be extended.

---

## Coding Style Requirements

This codebase follows **strict TDD**. You MUST:

1. **Write tests FIRST** before any implementation
2. Use **Go stdlib `testing` only** - no testify, gomega, etc.
3. Follow **Red-Green-Refactor**: failing test -> make it pass -> refactor
4. Use **table-driven tests** with `t.Run()`
5. Create **mock implementations** using function fields:
   ```go
   type mockPrompter struct {
       selectFunc func(prompt, def string, opts []string) (int, error)
   }
   ```
6. Add **compile-time interface checks**: `var _ Interface = (*Type)(nil)`
7. **Commit after each passing test** (pre-commit hook runs `go test ./...`)

### Existing Patterns to Follow

- Interfaces for DI: see `cmd.APIClient` interface
- Function field mocks: see `mockAPIClient` in `cmd/create_test.go`
- Error wrapping: `fmt.Errorf("context: %w", err)`
- Separation: `cmd/` for CLI, `internal/api/` for API calls

---

## Implementation Steps

### Step 1: Add `ListIssues` to API Layer

**File**: `internal/api/issues.go`

Add method to list open issues:

```go
type ListIssuesOptions struct {
    Owner   string
    Repo    string
    State   string // "open", "closed", "all"
    PerPage int
}

func (c *Client) ListIssues(opts ListIssuesOptions) ([]Issue, error)
```

**Test first** in `internal/api/issues_test.go`:
- Test successful list (mock HTTP 200 with JSON array)
- Test empty list
- Test API error

### Step 2: Extend `APIClient` Interface

**File**: `cmd/create.go`

Add to interface:
```go
type APIClient interface {
    CreateIssue(opts api.CreateIssueOptions) (*api.IssueResult, error)
    LinkSubIssue(opts api.LinkSubIssueOptions) error
    GetIssue(owner, repo string, number int) (*api.Issue, error)
    ListIssues(opts api.ListIssuesOptions) ([]api.Issue, error)  // NEW
}
```

Update `mockAPIClient` in tests.

### Step 3: Create Prompter Interface

**File**: `cmd/prompter.go` (new file)

```go
package cmd

// Prompter handles interactive user prompts.
type Prompter interface {
    Select(prompt string, defaultValue string, options []string) (int, error)
}
```

### Step 4: Implement Issue Selector

**File**: `cmd/select.go` (new file)

```go
package cmd

import (
    "fmt"
    "github.com/gwyn/gh-subissue/internal/api"
)

// SelectParentIssue prompts user to select an issue from a list.
// Returns the selected issue number.
func SelectParentIssue(p Prompter, issues []api.Issue) (int, error) {
    if len(issues) == 0 {
        return 0, fmt.Errorf("no open issues found")
    }

    options := make([]string, len(issues))
    for i, issue := range issues {
        // Format: "#123 Issue title" (truncate title if needed)
        title := issue.Title
        if len(title) > 50 {
            title = title[:47] + "..."
        }
        options[i] = fmt.Sprintf("#%d %s", issue.Number, title)
    }

    idx, err := p.Select("Select parent issue", "", options)
    if err != nil {
        return 0, err
    }

    return issues[idx].Number, nil
}
```

**Test first** in `cmd/select_test.go`:
- Test selection returns correct issue number
- Test empty issue list returns error
- Test prompter error propagates

### Step 5: Modify Flag Parsing

**File**: `cmd/create.go`

Change `ParseFlags` to allow `Parent == 0`:

```go
// Remove this validation from ParseFlags:
// if opts.Parent == 0 {
//     return nil, errors.New("--parent flag is required")
// }
```

Return `opts` with `Parent: 0` when flag not provided.

**Update tests** in `cmd/create_test.go`:
- Change "missing required parent flag" test to expect success with Parent=0
- Add new test case for "no parent flag returns zero"

### Step 6: Add Prompter to Runner

**File**: `cmd/create.go`

Extend `Runner`:

```go
type Runner struct {
    Client         APIClient
    Owner          string
    Repo           string
    Out            io.Writer
    Stdin          io.Reader
    ValidateParent bool
    OpenBrowser    func(url string) error
    Prompter       Prompter  // NEW - nil means non-interactive
}
```

### Step 7: Add Interactive Flow to Run()

**File**: `cmd/create.go`

At the start of `Run()`:

```go
func (r *Runner) Run(opts Options) error {
    // If no parent specified, prompt interactively
    if opts.Parent == 0 {
        if r.Prompter == nil {
            return errors.New("--parent flag is required (or run interactively)")
        }

        issues, err := r.Client.ListIssues(api.ListIssuesOptions{
            Owner:   r.Owner,
            Repo:    r.Repo,
            State:   "open",
            PerPage: 30,
        })
        if err != nil {
            return fmt.Errorf("failed to list issues: %w", err)
        }

        parent, err := SelectParentIssue(r.Prompter, issues)
        if err != nil {
            return err
        }
        opts.Parent = parent
    }

    // ... rest of existing Run() logic
}
```

**Test** in `cmd/create_test.go`:
- Test interactive selection when Parent=0 and Prompter set
- Test error when Parent=0 and Prompter nil
- Test API error during list propagates

### Step 8: Wire Up in main.go

**File**: `main.go`

```go
import (
    "github.com/cli/go-gh/v2/pkg/prompter"
    "github.com/cli/go-gh/v2/pkg/iostreams"
)

func runCreate(args []string) error {
    // ... existing code ...

    // Set up prompter for interactive mode
    ios := iostreams.System()
    var p cmd.Prompter
    if ios.IsStdinTTY() && ios.IsStdoutTTY() {
        p = prompter.New(ios.In, ios.Out, ios.ErrOut)
    }

    runner := &cmd.Runner{
        Client:         client,
        Owner:          owner,
        Repo:           repoName,
        Out:            os.Stdout,
        Stdin:          os.Stdin,
        ValidateParent: false,
        OpenBrowser:    b.Browse,
        Prompter:       p,  // NEW
    }

    return runner.Run(*opts)
}
```

### Step 9: Update Usage Text

**File**: `main.go`

Change help text:

```
  -p, --parent <number>    Parent issue number (interactive if omitted)
```

---

## Test Order

Execute in this order, committing after each green test:

1. `TestListIssues` in `internal/api/issues_test.go`
2. `TestSelectParentIssue` in `cmd/select_test.go`
3. Update `TestParseFlags` for no-parent case
4. `TestRunInteractiveSelection` in `cmd/create_test.go`
5. `TestRunNoPrompterRequiresParent` in `cmd/create_test.go`

---

## Dependencies

No new dependencies required. `go-gh/v2` already includes `pkg/prompter`.

Verify with:
```bash
go doc github.com/cli/go-gh/v2/pkg/prompter
```

---

## Notes for Implementation

- The `prompter.New()` from go-gh returns a concrete type, but our interface only needs `Select()`, so it will satisfy our minimal interface
- If `prompter` package is unavailable or API differs, research with: search the go-gh repository on GitHub for "prompter" usage examples
- The pre-commit hook runs tests automatically - commit frequently after each green test
- Keep changes minimal - don't refactor unrelated code
