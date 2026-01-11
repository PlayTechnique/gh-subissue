package cmd

import (
	"flag"
	"fmt"
	"io"
	"strconv"

	"github.com/gwyn/gh-subissue/internal/api"
	"github.com/gwyn/gh-subissue/internal/debug"
)

// EditOptions contains the parsed command line options for the edit command.
type EditOptions struct {
	IssueNumber int
	Project     OptionalString
	Repo        string
}

// ParseEditFlags parses command line flags for the edit command.
func ParseEditFlags(args []string) (*EditOptions, error) {
	debug.Log("ParseEditFlags", "args", args)

	opts := &EditOptions{}
	fs := flag.NewFlagSet("edit", flag.ContinueOnError)

	fs.Var(&opts.Project, "project", "Add to project (interactive if empty)")
	fs.Var(&opts.Project, "P", "Add to project (interactive if empty)")

	fs.StringVar(&opts.Repo, "repo", "", "Repository (owner/repo)")
	fs.StringVar(&opts.Repo, "R", "", "Repository (owner/repo)")

	// Parse to extract flags, but we need the issue number first
	if len(args) == 0 {
		return nil, fmt.Errorf("issue number is required")
	}

	// First arg should be the issue number
	issueNum, err := strconv.Atoi(args[0])
	if err != nil {
		return nil, fmt.Errorf("invalid issue number: %s", args[0])
	}
	opts.IssueNumber = issueNum

	// Parse remaining flags
	if len(args) > 1 {
		if err := fs.Parse(args[1:]); err != nil {
			debug.Error("ParseEditFlags", err, "stage", "fs.Parse")
			return nil, err
		}
	}

	debug.Log("ParseEditFlags", "parsed", fmt.Sprintf("%+v", opts))
	return opts, nil
}

// EditAPIClient defines the interface for edit operations.
type EditAPIClient interface {
	ListProjects(owner, repo string) ([]api.Project, error)
	GetIssueNodeID(owner, repo string, number int) (string, error)
	AddIssueToProject(projectID, issueNodeID string) error
}

// EditRunner executes the edit subcommand.
type EditRunner struct {
	Client   EditAPIClient
	Owner    string
	Repo     string
	Out      io.Writer
	Prompter Prompter
}

// Run executes the edit command.
func (r *EditRunner) Run(opts EditOptions) error {
	debug.Log("EditRunner.Run", "issue", opts.IssueNumber, "project", opts.Project.Value, "project_was_set", opts.Project.WasSet)

	// Currently only project assignment is supported
	if !opts.Project.WasSet {
		return fmt.Errorf("no edit options specified (use --project to add to a project)")
	}

	// List projects
	projects, err := r.Client.ListProjects(r.Owner, r.Repo)
	if err != nil {
		debug.Error("EditRunner.Run", err, "stage", "list_projects")
		return fmt.Errorf("failed to list projects: %w", err)
	}

	var selectedProject *api.Project

	if opts.Project.Value == "" {
		// Interactive mode
		if r.Prompter == nil {
			// List available projects in error message
			if len(projects) == 0 {
				return fmt.Errorf("no projects found for this repository\nCreate a project at: https://github.com/%s/%s/projects", r.Owner, r.Repo)
			}
			var names []string
			for _, p := range projects {
				names = append(names, p.Title)
			}
			return fmt.Errorf("--project requires a project name when not running interactively\nAvailable projects: %v\nExample: gh subissue edit %d --project %q", names, opts.IssueNumber, projects[0].Title)
		}

		if len(projects) == 0 {
			return fmt.Errorf("no projects found for this repository\nCreate a project at: https://github.com/%s/%s/projects", r.Owner, r.Repo)
		}

		project, err := SelectProject(r.Prompter, projects)
		if err != nil {
			debug.Error("EditRunner.Run", err, "stage", "select_project")
			return err
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
			var names []string
			for _, p := range projects {
				names = append(names, p.Title)
			}
			return fmt.Errorf("project %q not found\nAvailable projects: %v", opts.Project.Value, names)
		}
	}

	// Get issue node ID
	nodeID, err := r.Client.GetIssueNodeID(r.Owner, r.Repo, opts.IssueNumber)
	if err != nil {
		debug.Error("EditRunner.Run", err, "stage", "get_issue_node_id")
		return fmt.Errorf("failed to get issue: %w", err)
	}

	// Add to project
	err = r.Client.AddIssueToProject(selectedProject.ID, nodeID)
	if err != nil {
		debug.Error("EditRunner.Run", err, "stage", "add_issue_to_project")
		return fmt.Errorf("failed to add issue to project: %w", err)
	}

	fmt.Fprintf(r.Out, "Added issue #%d to project %q\n", opts.IssueNumber, selectedProject.Title)
	debug.Log("EditRunner.Run", "result", "success", "project", selectedProject.Title)
	return nil
}
