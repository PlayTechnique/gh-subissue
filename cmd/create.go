package cmd

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/gwyn/gh-subissue/internal/api"
)

// Options contains the parsed command line options.
type Options struct {
	Parent    int
	Title     string
	Body      string
	BodyFile  string
	Repo      string
	Assignees []string
	Labels    []string
	Milestone int
	Web       bool
}

// stringSlice is a flag.Value that collects multiple string values.
type stringSlice []string

func (s *stringSlice) String() string {
	return strings.Join(*s, ", ")
}

func (s *stringSlice) Set(value string) error {
	*s = append(*s, value)
	return nil
}

// ParseFlags parses command line arguments and returns Options.
func ParseFlags(args []string) (*Options, error) {
	fs := flag.NewFlagSet("create", flag.ContinueOnError)

	opts := &Options{}
	var assignees, labels stringSlice

	fs.IntVar(&opts.Parent, "parent", 0, "Parent issue number (required)")
	fs.IntVar(&opts.Parent, "p", 0, "Parent issue number (required)")

	fs.StringVar(&opts.Title, "title", "", "Issue title")
	fs.StringVar(&opts.Title, "t", "", "Issue title")

	fs.StringVar(&opts.Body, "body", "", "Issue body")
	fs.StringVar(&opts.Body, "b", "", "Issue body")

	fs.StringVar(&opts.BodyFile, "body-file", "", "Read body from file (use - for stdin)")

	fs.StringVar(&opts.Repo, "repo", "", "Repository in owner/repo format")
	fs.StringVar(&opts.Repo, "R", "", "Repository in owner/repo format")

	fs.Var(&assignees, "assignee", "Assign users (can be repeated)")
	fs.Var(&assignees, "a", "Assign users (can be repeated)")

	fs.Var(&labels, "label", "Add labels (can be repeated)")
	fs.Var(&labels, "l", "Add labels (can be repeated)")

	fs.IntVar(&opts.Milestone, "milestone", 0, "Milestone number")
	fs.IntVar(&opts.Milestone, "m", 0, "Milestone number")

	fs.BoolVar(&opts.Web, "web", false, "Open in browser after creation")
	fs.BoolVar(&opts.Web, "w", false, "Open in browser after creation")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	opts.Assignees = assignees
	opts.Labels = labels

	if opts.Parent == 0 {
		return nil, errors.New("--parent flag is required")
	}

	return opts, nil
}

// ParseRepo parses an owner/repo string into owner and repo parts.
func ParseRepo(s string) (owner, repo string, err error) {
	if s == "" {
		return "", "", errors.New("repository cannot be empty")
	}

	parts := strings.Split(s, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid repository format: %q (expected owner/repo)", s)
	}

	return parts[0], parts[1], nil
}

// ReadBody reads the issue body from a file or stdin.
func ReadBody(path string, stdin io.Reader) (string, error) {
	if path == "-" {
		if stdin == nil {
			return "", errors.New("stdin is nil")
		}
		data, err := io.ReadAll(stdin)
		if err != nil {
			return "", fmt.Errorf("failed to read from stdin: %w", err)
		}
		return string(data), nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read file %q: %w", path, err)
	}
	return string(data), nil
}

// APIClient defines the interface for GitHub API operations.
type APIClient interface {
	CreateIssue(opts api.CreateIssueOptions) (*api.IssueResult, error)
	LinkSubIssue(opts api.LinkSubIssueOptions) error
	GetIssue(owner, repo string, number int) (*api.Issue, error)
}

// Runner executes the create subcommand.
type Runner struct {
	Client         APIClient
	Owner          string
	Repo           string
	Out            io.Writer
	Stdin          io.Reader
	ValidateParent bool
	OpenBrowser    func(url string) error
}

// Run executes the create command with the given options.
func (r *Runner) Run(opts Options) error {
	// Read body from file if specified
	body := opts.Body
	if opts.BodyFile != "" {
		var err error
		body, err = ReadBody(opts.BodyFile, r.Stdin)
		if err != nil {
			return err
		}
	}

	// Validate parent exists if requested
	if r.ValidateParent {
		_, err := r.Client.GetIssue(r.Owner, r.Repo, opts.Parent)
		if err != nil {
			return fmt.Errorf("parent issue #%d not found: %w", opts.Parent, err)
		}
	}

	// Create the issue
	result, err := r.Client.CreateIssue(api.CreateIssueOptions{
		Owner:     r.Owner,
		Repo:      r.Repo,
		Title:     opts.Title,
		Body:      body,
		Labels:    opts.Labels,
		Assignees: opts.Assignees,
		Milestone: opts.Milestone,
	})
	if err != nil {
		return fmt.Errorf("failed to create issue: %w", err)
	}

	// Link as sub-issue
	linkErr := r.Client.LinkSubIssue(api.LinkSubIssueOptions{
		Owner:       r.Owner,
		Repo:        r.Repo,
		ParentIssue: opts.Parent,
		SubIssueID:  result.ID,
	})

	if linkErr != nil {
		// Issue was created but linking failed - warn the user
		fmt.Fprintf(r.Out, "Warning: Issue created but failed to link as sub-issue: %v\n", linkErr)
		fmt.Fprintf(r.Out, "Issue URL: %s\n", result.URL)
		fmt.Fprintf(r.Out, "To manually link, run:\n")
		fmt.Fprintf(r.Out, "  gh api repos/%s/%s/issues/%d/sub_issues -f sub_issue_id=%d\n",
			r.Owner, r.Repo, opts.Parent, result.ID)
		return nil
	}

	fmt.Fprintln(r.Out, result.URL)

	// Open in browser if requested
	if opts.Web && r.OpenBrowser != nil {
		if err := r.OpenBrowser(result.URL); err != nil {
			fmt.Fprintf(r.Out, "Warning: failed to open browser: %v\n", err)
		}
	}

	return nil
}
