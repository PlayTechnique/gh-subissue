package cmd

import (
	"flag"
	"fmt"
	"io"

	"github.com/gwyn/gh-subissue/internal/api"
	"github.com/gwyn/gh-subissue/internal/debug"
)

// ReposOptions contains the parsed command line options for the repos command.
type ReposOptions struct {
	Owner    string
	Limit    int
	Enabled  bool
	Disabled bool
	NoHeader bool
}

// ParseReposFlags parses command line flags for the repos command.
func ParseReposFlags(args []string) (*ReposOptions, error) {
	debug.Log("ParseReposFlags", "args", args)

	opts := &ReposOptions{}
	fs := flag.NewFlagSet("repos", flag.ContinueOnError)

	fs.IntVar(&opts.Limit, "limit", 30, "Maximum repos to list")
	fs.IntVar(&opts.Limit, "L", 30, "Maximum repos to list")

	fs.BoolVar(&opts.Enabled, "enabled", false, "Show only repos where sub-issues work")
	fs.BoolVar(&opts.Disabled, "disabled", false, "Show only repos where sub-issues don't work")
	fs.BoolVar(&opts.NoHeader, "no-header", false, "Omit table header from output")

	// Extract positional owner arg before flags (if present)
	// Go's flag package stops at the first non-flag, so we need to
	// reorder args to put flags first
	var owner string
	var flagArgs []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			flagArgs = append(flagArgs, args[i:]...)
			break
		}
		if len(arg) > 0 && arg[0] == '-' {
			flagArgs = append(flagArgs, arg)
			// Check if this flag takes a value
			if arg == "-L" || arg == "--limit" {
				if i+1 < len(args) {
					i++
					flagArgs = append(flagArgs, args[i])
				}
			}
		} else if owner == "" {
			owner = arg
		}
	}
	opts.Owner = owner

	if err := fs.Parse(flagArgs); err != nil {
		debug.Error("ParseReposFlags", err, "stage", "fs.Parse")
		return nil, err
	}

	debug.Log("ParseReposFlags", "parsed", fmt.Sprintf("%+v", opts))
	return opts, nil
}

// ReposAPIClient defines the interface for repos operations.
type ReposAPIClient interface {
	ListRepositories(opts api.ListRepositoriesOptions) ([]api.Repository, error)
	GetAuthenticatedUser() (*api.User, error)
}

// ReposRunner executes the repos subcommand.
type ReposRunner struct {
	Client ReposAPIClient
	Out    io.Writer
}

// Run executes the repos command.
func (r *ReposRunner) Run(opts ReposOptions) error {
	debug.Log("ReposRunner.Run", "owner", opts.Owner, "limit", opts.Limit, "enabled", opts.Enabled, "disabled", opts.Disabled)

	owner := opts.Owner

	// If no owner specified, use authenticated user
	if owner == "" {
		user, err := r.Client.GetAuthenticatedUser()
		if err != nil {
			debug.Error("ReposRunner.Run", err, "stage", "get_authenticated_user")
			return fmt.Errorf("could not determine user: %w\n\nSpecify an owner:\n  gh subissue repos <owner>", err)
		}
		owner = user.Login
		debug.Log("ReposRunner.Run", "resolved_owner", owner)
	}

	// Fetch repositories (may need pagination for large limits)
	var allRepos []api.Repository
	perPage := 100
	if opts.Limit < perPage {
		perPage = opts.Limit
	}

	page := 1
	for len(allRepos) < opts.Limit {
		repos, err := r.Client.ListRepositories(api.ListRepositoriesOptions{
			Owner:   owner,
			PerPage: perPage,
			Page:    page,
		})
		if err != nil {
			debug.Error("ReposRunner.Run", err, "stage", "list_repositories")
			return err
		}

		if len(repos) == 0 {
			break
		}

		allRepos = append(allRepos, repos...)
		page++

		// If we got fewer than requested, no more pages
		if len(repos) < perPage {
			break
		}
	}

	// Truncate to limit
	if len(allRepos) > opts.Limit {
		allRepos = allRepos[:opts.Limit]
	}

	// Filter repos based on flags
	var filtered []api.Repository
	for _, repo := range allRepos {
		enabled := repo.HasIssues && !repo.Archived
		if opts.Enabled && !enabled {
			continue
		}
		if opts.Disabled && enabled {
			continue
		}
		filtered = append(filtered, repo)
	}

	if len(filtered) == 0 {
		if opts.Enabled || opts.Disabled {
			fmt.Fprintf(r.Out, "No matching repositories found for %s\n", owner)
		} else {
			fmt.Fprintf(r.Out, "No repositories found for %s\n", owner)
		}
		return nil
	}

	// Print header and repos
	if !opts.NoHeader {
		fmt.Fprintf(r.Out, "%-19s %s\n", "REPOSITORY", "SUB-ISSUES")
	}
	for _, repo := range filtered {
		status := repoStatus(repo)
		fmt.Fprintf(r.Out, "%-19s %s\n", repo.FullName, status)
	}

	return nil
}

// repoStatus returns a human-readable status for sub-issue support.
func repoStatus(repo api.Repository) string {
	if repo.Archived {
		return "disabled (archived)"
	}
	if !repo.HasIssues {
		return "disabled (issues off)"
	}
	return "enabled"
}
