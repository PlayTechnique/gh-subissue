package cmd

import (
	"flag"
	"fmt"
	"io"

	"github.com/gwyn/gh-subissue/internal/api"
	"github.com/gwyn/gh-subissue/internal/debug"
)

// ListOptions contains the parsed command line options for the list command.
type ListOptions struct {
	Parent int
	Repo   string
}

// ParseListFlags parses command line flags for the list command.
func ParseListFlags(args []string) (*ListOptions, error) {
	debug.Log("ParseListFlags", "args", args)

	opts := &ListOptions{}
	fs := flag.NewFlagSet("list", flag.ContinueOnError)

	fs.IntVar(&opts.Parent, "parent", 0, "Parent issue number")
	fs.IntVar(&opts.Parent, "p", 0, "Parent issue number")

	fs.StringVar(&opts.Repo, "repo", "", "Repository (owner/repo)")
	fs.StringVar(&opts.Repo, "R", "", "Repository (owner/repo)")

	if err := fs.Parse(args); err != nil {
		debug.Error("ParseListFlags", err, "stage", "fs.Parse")
		return nil, err
	}

	debug.Log("ParseListFlags", "parsed", fmt.Sprintf("%+v", opts))
	return opts, nil
}

// ListAPIClient defines the interface for list operations.
type ListAPIClient interface {
	ListSubIssues(opts api.ListSubIssuesOptions) ([]api.Issue, error)
	ListIssues(opts api.ListIssuesOptions) ([]api.Issue, error)
}

// ListRunner executes the list subcommand.
type ListRunner struct {
	Client   ListAPIClient
	Owner    string
	Repo     string
	Out      io.Writer
	Prompter Prompter
}

// Run executes the list command.
func (r *ListRunner) Run(opts ListOptions) error {
	debug.Log("ListRunner.Run", "parent", opts.Parent)

	parent := opts.Parent

	// If no parent specified, prompt for selection
	if parent == 0 {
		if r.Prompter == nil {
			return fmt.Errorf("--parent flag is required when not running interactively")
		}

		issues, err := r.Client.ListIssues(api.ListIssuesOptions{
			Owner:   r.Owner,
			Repo:    r.Repo,
			State:   "open",
			PerPage: 30,
		})
		if err != nil {
			debug.Error("ListRunner.Run", err, "stage", "list_issues")
			return err
		}

		selected, err := SelectParentIssue(r.Prompter, issues)
		if err != nil {
			debug.Error("ListRunner.Run", err, "stage", "select_parent")
			return err
		}
		parent = selected
	}

	// List sub-issues
	subIssues, err := r.Client.ListSubIssues(api.ListSubIssuesOptions{
		Owner:       r.Owner,
		Repo:        r.Repo,
		ParentIssue: parent,
	})
	if err != nil {
		debug.Error("ListRunner.Run", err, "stage", "list_sub_issues")
		return err
	}

	if len(subIssues) == 0 {
		fmt.Fprintf(r.Out, "No sub-issues found for issue #%d\n", parent)
		return nil
	}

	for _, issue := range subIssues {
		fmt.Fprintf(r.Out, "#%d\t%s\n", issue.Number, issue.Title)
	}

	return nil
}
