package cmd

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/gwyn/gh-subissue/internal/api"
	"github.com/gwyn/gh-subissue/internal/debug"
)

// OptionalString is a flag type that tracks whether it was explicitly set.
// This allows distinguishing between "not provided", "provided empty", and "provided with value".
type OptionalString struct {
	Value  string
	WasSet bool
}

func (o *OptionalString) String() string {
	return o.Value
}

func (o *OptionalString) Set(value string) error {
	o.Value = value
	o.WasSet = true
	return nil
}

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
	Project   OptionalString
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
	debug.Log("ParseFlags", "args", args)

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

	fs.Var(&opts.Project, "project", "Add to project (interactive if empty)")
	fs.Var(&opts.Project, "P", "Add to project (interactive if empty)")

	if err := fs.Parse(args); err != nil {
		debug.Error("ParseFlags", err, "stage", "fs.Parse")
		return nil, err
	}

	opts.Assignees = assignees
	opts.Labels = labels

	debug.Log("ParseFlags", "parsed_parent", opts.Parent, "title", opts.Title, "repo", opts.Repo)
	return opts, nil
}

// ParseRepo parses an owner/repo string into owner and repo parts.
func ParseRepo(s string) (owner, repo string, err error) {
	debug.Log("ParseRepo", "input", s)

	if s == "" {
		err := errors.New("repository cannot be empty")
		debug.Error("ParseRepo", err)
		return "", "", err
	}

	parts := strings.Split(s, "/")
	if len(parts) != 2 {
		err := fmt.Errorf("invalid repository format: %q (expected owner/repo)", s)
		debug.Error("ParseRepo", err)
		return "", "", err
	}

	debug.Log("ParseRepo", "owner", parts[0], "repo", parts[1])
	return parts[0], parts[1], nil
}

// ReadBody reads the issue body from a file or stdin.
func ReadBody(path string, stdin io.Reader) (string, error) {
	debug.Log("ReadBody", "path", path, "stdin_nil", stdin == nil)

	if path == "-" {
		if stdin == nil {
			err := errors.New("stdin is nil")
			debug.Error("ReadBody", err)
			return "", err
		}
		debug.Log("ReadBody", "source", "stdin")
		data, err := io.ReadAll(stdin)
		if err != nil {
			debug.Error("ReadBody", err, "stage", "read_stdin")
			return "", fmt.Errorf("failed to read from stdin: %w", err)
		}
		debug.Log("ReadBody", "bytes_read", len(data))
		return string(data), nil
	}

	debug.Log("ReadBody", "source", "file", "file_path", path)
	data, err := os.ReadFile(path)
	if err != nil {
		debug.Error("ReadBody", err, "stage", "read_file")
		return "", fmt.Errorf("failed to read file %q: %w", path, err)
	}
	debug.Log("ReadBody", "bytes_read", len(data))
	return string(data), nil
}

// APIClient defines the interface for GitHub API operations.
type APIClient interface {
	CreateIssue(opts api.CreateIssueOptions) (*api.IssueResult, error)
	LinkSubIssue(opts api.LinkSubIssueOptions) error
	GetIssue(owner, repo string, number int) (*api.Issue, error)
	ListIssues(opts api.ListIssuesOptions) ([]api.Issue, error)
	ListProjects(owner, repo string) ([]api.Project, error)
	GetIssueNodeID(owner, repo string, number int) (string, error)
	AddIssueToProject(projectID, issueNodeID string) error
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
	Prompter       Prompter // nil means non-interactive mode
}

// Run executes the create command with the given options.
func (r *Runner) Run(opts Options) error {
	debug.Log("Runner.Run", "owner", r.Owner, "repo", r.Repo, "parent", opts.Parent, "title", opts.Title, "has_prompter", r.Prompter != nil)

	// If no parent specified, prompt interactively
	if opts.Parent == 0 {
		debug.Log("Runner.Run", "action", "need_parent_selection")
		if r.Prompter == nil {
			err := errors.New("--parent flag is required when not running interactively\n\nTip: Run in a terminal for interactive parent selection, or use --parent to specify the parent issue number")
			debug.Error("Runner.Run", err, "reason", "no_prompter")
			return err
		}

		debug.Log("Runner.Run", "action", "listing_issues_for_selection")
		issues, err := r.Client.ListIssues(api.ListIssuesOptions{
			Owner:   r.Owner,
			Repo:    r.Repo,
			State:   "open",
			PerPage: 30,
		})
		if err != nil {
			debug.Error("Runner.Run", err, "stage", "list_issues")
			return fmt.Errorf("failed to list issues: %w", err)
		}

		debug.Log("Runner.Run", "issues_found", len(issues))
		parent, err := SelectParentIssue(r.Prompter, issues)
		if err != nil {
			debug.Error("Runner.Run", err, "stage", "select_parent")
			return err
		}
		opts.Parent = parent
		debug.Log("Runner.Run", "selected_parent", parent)
	}

	// If no title specified, prompt interactively
	if opts.Title == "" {
		debug.Log("Runner.Run", "action", "need_title_input")
		if r.Prompter == nil {
			err := errors.New("--title flag is required when not running interactively\n\nTip: Run in a terminal for interactive input, or use --title to specify the issue title")
			debug.Error("Runner.Run", err, "reason", "no_prompter")
			return err
		}

		debug.Log("Runner.Run", "action", "prompting_for_title")
		title, err := r.Prompter.Input("Title", "")
		if err != nil {
			debug.Error("Runner.Run", err, "stage", "input_title")
			return fmt.Errorf("failed to get title: %w", err)
		}

		title = strings.TrimSpace(title)
		if title == "" {
			err := errors.New("title cannot be empty")
			debug.Error("Runner.Run", err, "reason", "empty_title")
			return err
		}

		opts.Title = title
		debug.Log("Runner.Run", "entered_title", title)
	}

	// Read body from file if specified
	body := opts.Body
	if opts.BodyFile != "" {
		debug.Log("Runner.Run", "action", "reading_body_file", "body_file", opts.BodyFile)
		var err error
		body, err = ReadBody(opts.BodyFile, r.Stdin)
		if err != nil {
			debug.Error("Runner.Run", err, "stage", "read_body")
			return err
		}
	}

	// Validate parent exists if requested
	if r.ValidateParent {
		debug.Log("Runner.Run", "action", "validating_parent", "parent", opts.Parent)
		_, err := r.Client.GetIssue(r.Owner, r.Repo, opts.Parent)
		if err != nil {
			debug.Error("Runner.Run", err, "stage", "validate_parent")
			return fmt.Errorf("parent issue #%d not found: %w", opts.Parent, err)
		}
		debug.Log("Runner.Run", "parent_validation", "success")
	}

	// Create the issue
	debug.Log("Runner.Run", "action", "creating_issue", "title", opts.Title)
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
		debug.Error("Runner.Run", err, "stage", "create_issue")
		return err // APIError already has good context
	}
	debug.Log("Runner.Run", "issue_created", result.Number, "issue_id", result.ID, "url", result.URL)

	// Link as sub-issue
	debug.Log("Runner.Run", "action", "linking_sub_issue", "parent", opts.Parent, "sub_issue_id", result.ID)
	linkErr := r.Client.LinkSubIssue(api.LinkSubIssueOptions{
		Owner:       r.Owner,
		Repo:        r.Repo,
		ParentIssue: opts.Parent,
		SubIssueID:  result.ID,
	})

	if linkErr != nil {
		// Issue was created but linking failed - warn the user
		debug.Error("Runner.Run", linkErr, "stage", "link_sub_issue", "issue_url", result.URL)
		fmt.Fprintf(r.Out, "Warning: Issue created but failed to link as sub-issue: %v\n", linkErr)
		fmt.Fprintf(r.Out, "Issue URL: %s\n", result.URL)
		fmt.Fprintf(r.Out, "To manually link, run:\n")
		fmt.Fprintf(r.Out, "  gh api repos/%s/%s/issues/%d/sub_issues -f sub_issue_id=%d\n",
			r.Owner, r.Repo, opts.Parent, result.ID)
		return nil
	}

	// Add to project if requested
	if opts.Project.WasSet {
		debug.Log("Runner.Run", "action", "adding_to_project", "project_value", opts.Project.Value)
		r.addToProject(opts, result)
	}

	debug.Log("Runner.Run", "result", "success", "url", result.URL)
	fmt.Fprintln(r.Out, result.URL)

	// Open in browser if requested
	if opts.Web && r.OpenBrowser != nil {
		debug.Log("Runner.Run", "action", "opening_browser", "url", result.URL)
		if err := r.OpenBrowser(result.URL); err != nil {
			debug.Error("Runner.Run", err, "stage", "open_browser")
			fmt.Fprintf(r.Out, "Warning: failed to open browser: %v\n", err)
		}
	}

	return nil
}

// addToProject handles adding the created issue to a project.
func (r *Runner) addToProject(opts Options, result *api.IssueResult) {
	// List projects
	projects, err := r.Client.ListProjects(r.Owner, r.Repo)
	if err != nil {
		debug.Error("addToProject", err, "stage", "list_projects")
		fmt.Fprintf(r.Out, "Warning: failed to list projects: %v\n", err)
		return
	}

	var selectedProject *api.Project

	if opts.Project.Value == "" {
		// Interactive mode - prompt user to select
		if r.Prompter == nil {
			debug.Log("addToProject", "action", "skip_interactive", "reason", "no_prompter")
			fmt.Fprintf(r.Out, "Warning: --project requires a project name when not running interactively\n")
			return
		}

		if len(projects) == 0 {
			debug.Log("addToProject", "action", "no_projects_found")
			fmt.Fprintf(r.Out, "Warning: no projects found for this repository\n")
			return
		}

		project, err := SelectProject(r.Prompter, projects)
		if err != nil {
			debug.Error("addToProject", err, "stage", "select_project")
			fmt.Fprintf(r.Out, "Warning: failed to select project: %v\n", err)
			return
		}
		selectedProject = project
	} else {
		// Find project by name
		for i := range projects {
			if projects[i].Title == opts.Project.Value {
				selectedProject = &projects[i]
				break
			}
		}
		if selectedProject == nil {
			debug.Log("addToProject", "action", "project_not_found", "project_name", opts.Project.Value)
			fmt.Fprintf(r.Out, "Warning: project %q not found\n", opts.Project.Value)
			return
		}
	}

	// Get issue node ID for GraphQL
	nodeID, err := r.Client.GetIssueNodeID(r.Owner, r.Repo, result.Number)
	if err != nil {
		debug.Error("addToProject", err, "stage", "get_issue_node_id")
		fmt.Fprintf(r.Out, "Warning: failed to get issue node ID: %v\n", err)
		return
	}

	// Add issue to project
	err = r.Client.AddIssueToProject(selectedProject.ID, nodeID)
	if err != nil {
		debug.Error("addToProject", err, "stage", "add_issue_to_project")
		fmt.Fprintf(r.Out, "Warning: failed to add issue to project: %v\n", err)
		return
	}

	debug.Log("addToProject", "result", "success", "project", selectedProject.Title)
}
