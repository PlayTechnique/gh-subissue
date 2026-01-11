package main

import (
	"fmt"
	"os"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/browser"
	"github.com/cli/go-gh/v2/pkg/prompter"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/cli/go-gh/v2/pkg/term"

	"github.com/gwyn/gh-subissue/cmd"
	internalapi "github.com/gwyn/gh-subissue/internal/api"
	"github.com/gwyn/gh-subissue/internal/debug"
)

func main() {
	debug.Init()
	debug.Log("main", "version", "0.1.0", "args", os.Args)

	if err := run(); err != nil {
		debug.Error("main", err)
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	args := os.Args[1:]
	debug.Log("run", "subcommand_args", args)

	if len(args) == 0 {
		debug.Log("run", "action", "printUsage", "reason", "no_args")
		return printUsage()
	}

	switch args[0] {
	case "create":
		debug.Log("run", "action", "runCreate", "create_args", args[1:])
		return runCreate(args[1:])
	case "list":
		debug.Log("run", "action", "runList", "list_args", args[1:])
		return runList(args[1:])
	case "edit":
		debug.Log("run", "action", "runEdit", "edit_args", args[1:])
		return runEdit(args[1:])
	case "repos":
		debug.Log("run", "action", "runRepos", "repos_args", args[1:])
		return runRepos(args[1:])
	case "help", "--help", "-h":
		debug.Log("run", "action", "printUsage", "reason", "help_flag")
		return printUsage()
	case "version", "--version":
		debug.Log("run", "action", "printVersion")
		fmt.Println("gh-subissue version 0.1.0")
		return nil
	default:
		debug.Log("run", "action", "unknown_command", "command", args[0])
		return fmt.Errorf("unknown command: %s", args[0])
	}
}

func runCreate(args []string) error {
	debug.Log("runCreate", "args", args)

	opts, err := cmd.ParseFlags(args)
	if err != nil {
		debug.Error("runCreate", err, "stage", "ParseFlags")
		return err
	}
	debug.Log("runCreate", "parsed_opts", fmt.Sprintf("%+v", opts))

	// Set up prompter for interactive mode first (needed for repo resolution)
	// Use term.FromEnv() to respect GH_FORCE_TTY and other env vars
	var p cmd.Prompter
	t := term.FromEnv()
	isStdinTerminal := term.IsTerminal(os.Stdin)
	isOutputTerminal := t.IsTerminalOutput()
	debug.Log("runCreate", "stdin_is_terminal", isStdinTerminal, "output_is_terminal", isOutputTerminal)

	if isStdinTerminal && isOutputTerminal {
		p = prompter.New(os.Stdin, os.Stdout, os.Stderr)
		debug.Log("runCreate", "prompter", "enabled")
	} else {
		debug.Log("runCreate", "prompter", "disabled")
	}

	// Resolve repository
	var owner, repoName, host string
	if opts.Repo != "" {
		debug.Log("runCreate", "repo_source", "flag", "repo_flag", opts.Repo)
		owner, repoName, err = cmd.ParseRepo(opts.Repo)
		if err != nil {
			debug.Error("runCreate", err, "stage", "ParseRepo")
			return err
		}
		// When using --repo flag, try to detect host from current repo, fallback to github.com
		if repo, err := repository.Current(); err == nil {
			host = repo.Host
			debug.Log("runCreate", "host_source", "current_repo", "host", host)
		} else {
			host = "github.com"
			debug.Log("runCreate", "host_source", "fallback", "host", host)
		}
	} else {
		debug.Log("runCreate", "repo_source", "current_directory")
		repo, repoErr := repository.Current()
		if repoErr != nil {
			debug.Log("runCreate", "repo_lookup_failed", repoErr.Error())
			// Try interactive prompt if available
			if p != nil {
				debug.Log("runCreate", "action", "prompting_for_repo")
				owner, repoName, err = cmd.PromptRepository(p)
				if err != nil {
					debug.Error("runCreate", err, "stage", "PromptRepository")
					return err
				}
				host = "github.com"
			} else {
				debug.Error("runCreate", repoErr, "stage", "repository.Current")
				return fmt.Errorf("could not determine repository: %w\n\nTo list your repositories:\n  gh repo list\n\nThen specify with --repo:\n  gh subissue create --repo owner/repo", repoErr)
			}
		} else {
			owner = repo.Owner
			repoName = repo.Name
			host = repo.Host
			debug.Log("runCreate", "resolved_repo", owner+"/"+repoName, "host", host)
		}
	}

	// Determine API base URL based on host
	var baseURL string
	if host == "github.com" || host == "" {
		baseURL = "https://api.github.com"
	} else {
		baseURL = fmt.Sprintf("https://%s/api/v3", host)
	}
	debug.Log("runCreate", "base_url", baseURL, "owner", owner, "repo", repoName)

	// Create authenticated HTTP client from go-gh
	httpClient, err := api.DefaultHTTPClient()
	if err != nil {
		debug.Error("runCreate", err, "stage", "DefaultHTTPClient")
		return fmt.Errorf("failed to create HTTP client: %w", err)
	}
	debug.Log("runCreate", "http_client", "created")

	// Use the tested internal/api.Client
	client := &internalapi.Client{
		HTTPClient: httpClient,
		BaseURL:    baseURL,
	}

	// Set up browser opener
	b := browser.New("", os.Stdout, os.Stderr)

	runner := &cmd.Runner{
		Client:         client,
		Owner:          owner,
		Repo:           repoName,
		Out:            os.Stdout,
		Stdin:          os.Stdin,
		ValidateParent: false,
		OpenBrowser:    b.Browse,
		Prompter:       p,
	}

	debug.Log("runCreate", "action", "runner.Run", "owner", owner, "repo", repoName)
	return runner.Run(*opts)
}

func runList(args []string) error {
	debug.Log("runList", "args", args)

	opts, err := cmd.ParseListFlags(args)
	if err != nil {
		debug.Error("runList", err, "stage", "ParseListFlags")
		return err
	}
	debug.Log("runList", "parsed_opts", fmt.Sprintf("%+v", opts))

	// Set up prompter for interactive mode first (needed for repo resolution)
	var p cmd.Prompter
	t := term.FromEnv()
	isStdinTerminal := term.IsTerminal(os.Stdin)
	isOutputTerminal := t.IsTerminalOutput()
	debug.Log("runList", "stdin_is_terminal", isStdinTerminal, "output_is_terminal", isOutputTerminal)

	if isStdinTerminal && isOutputTerminal {
		p = prompter.New(os.Stdin, os.Stdout, os.Stderr)
		debug.Log("runList", "prompter", "enabled")
	} else {
		debug.Log("runList", "prompter", "disabled")
	}

	// Resolve repository
	var owner, repoName, host string
	if opts.Repo != "" {
		debug.Log("runList", "repo_source", "flag", "repo_flag", opts.Repo)
		owner, repoName, err = cmd.ParseRepo(opts.Repo)
		if err != nil {
			debug.Error("runList", err, "stage", "ParseRepo")
			return err
		}
		if repo, err := repository.Current(); err == nil {
			host = repo.Host
		} else {
			host = "github.com"
		}
	} else {
		debug.Log("runList", "repo_source", "current_directory")
		repo, repoErr := repository.Current()
		if repoErr != nil {
			debug.Log("runList", "repo_lookup_failed", repoErr.Error())
			// Try interactive prompt if available
			if p != nil {
				debug.Log("runList", "action", "prompting_for_repo")
				owner, repoName, err = cmd.PromptRepository(p)
				if err != nil {
					debug.Error("runList", err, "stage", "PromptRepository")
					return err
				}
				host = "github.com"
			} else {
				debug.Error("runList", repoErr, "stage", "repository.Current")
				return fmt.Errorf("could not determine repository: %w\n\nTo list your repositories:\n  gh repo list\n\nThen specify with --repo:\n  gh subissue list --repo owner/repo", repoErr)
			}
		} else {
			owner = repo.Owner
			repoName = repo.Name
			host = repo.Host
		}
	}

	// Determine API base URL based on host
	var baseURL string
	if host == "github.com" || host == "" {
		baseURL = "https://api.github.com"
	} else {
		baseURL = fmt.Sprintf("https://%s/api/v3", host)
	}

	// Create authenticated HTTP client
	httpClient, err := api.DefaultHTTPClient()
	if err != nil {
		debug.Error("runList", err, "stage", "DefaultHTTPClient")
		return fmt.Errorf("failed to create HTTP client: %w", err)
	}

	client := &internalapi.Client{
		HTTPClient: httpClient,
		BaseURL:    baseURL,
	}

	runner := &cmd.ListRunner{
		Client:   client,
		Owner:    owner,
		Repo:     repoName,
		Out:      os.Stdout,
		Prompter: p,
	}

	return runner.Run(*opts)
}

func runEdit(args []string) error {
	debug.Log("runEdit", "args", args)

	opts, err := cmd.ParseEditFlags(args)
	if err != nil {
		debug.Error("runEdit", err, "stage", "ParseEditFlags")
		return err
	}
	debug.Log("runEdit", "parsed_opts", fmt.Sprintf("%+v", opts))

	// Set up prompter for interactive mode first (needed for repo resolution)
	var p cmd.Prompter
	t := term.FromEnv()
	isStdinTerminal := term.IsTerminal(os.Stdin)
	isOutputTerminal := t.IsTerminalOutput()
	debug.Log("runEdit", "stdin_is_terminal", isStdinTerminal, "output_is_terminal", isOutputTerminal)

	if isStdinTerminal && isOutputTerminal {
		p = prompter.New(os.Stdin, os.Stdout, os.Stderr)
		debug.Log("runEdit", "prompter", "enabled")
	} else {
		debug.Log("runEdit", "prompter", "disabled")
	}

	// Resolve repository
	var owner, repoName, host string
	if opts.Repo != "" {
		debug.Log("runEdit", "repo_source", "flag", "repo_flag", opts.Repo)
		owner, repoName, err = cmd.ParseRepo(opts.Repo)
		if err != nil {
			debug.Error("runEdit", err, "stage", "ParseRepo")
			return err
		}
		if repo, err := repository.Current(); err == nil {
			host = repo.Host
		} else {
			host = "github.com"
		}
	} else {
		debug.Log("runEdit", "repo_source", "current_directory")
		repo, repoErr := repository.Current()
		if repoErr != nil {
			debug.Log("runEdit", "repo_lookup_failed", repoErr.Error())
			// Try interactive prompt if available
			if p != nil {
				debug.Log("runEdit", "action", "prompting_for_repo")
				owner, repoName, err = cmd.PromptRepository(p)
				if err != nil {
					debug.Error("runEdit", err, "stage", "PromptRepository")
					return err
				}
				host = "github.com"
			} else {
				debug.Error("runEdit", repoErr, "stage", "repository.Current")
				return fmt.Errorf("could not determine repository: %w\n\nTo list your repositories:\n  gh repo list\n\nThen specify with --repo:\n  gh subissue edit <issue-number> --repo owner/repo", repoErr)
			}
		} else {
			owner = repo.Owner
			repoName = repo.Name
			host = repo.Host
		}
	}

	// Determine API base URL based on host
	var baseURL string
	if host == "github.com" || host == "" {
		baseURL = "https://api.github.com"
	} else {
		baseURL = fmt.Sprintf("https://%s/api/v3", host)
	}

	// Create authenticated HTTP client
	httpClient, err := api.DefaultHTTPClient()
	if err != nil {
		debug.Error("runEdit", err, "stage", "DefaultHTTPClient")
		return fmt.Errorf("failed to create HTTP client: %w", err)
	}

	client := &internalapi.Client{
		HTTPClient: httpClient,
		BaseURL:    baseURL,
	}

	runner := &cmd.EditRunner{
		Client:   client,
		Owner:    owner,
		Repo:     repoName,
		Out:      os.Stdout,
		Prompter: p,
	}

	return runner.Run(*opts)
}

func runRepos(args []string) error {
	debug.Log("runRepos", "args", args)

	opts, err := cmd.ParseReposFlags(args)
	if err != nil {
		debug.Error("runRepos", err, "stage", "ParseReposFlags")
		return err
	}
	debug.Log("runRepos", "parsed_opts", fmt.Sprintf("%+v", opts))

	// Create authenticated HTTP client
	httpClient, err := api.DefaultHTTPClient()
	if err != nil {
		debug.Error("runRepos", err, "stage", "DefaultHTTPClient")
		return fmt.Errorf("failed to create HTTP client: %w", err)
	}

	// Use github.com as the default host for repos command
	baseURL := "https://api.github.com"

	client := &internalapi.Client{
		HTTPClient: httpClient,
		BaseURL:    baseURL,
	}

	runner := &cmd.ReposRunner{
		Client: client,
		Out:    os.Stdout,
	}

	return runner.Run(*opts)
}

func printUsage() error {
	usage := `gh-subissue - Create and manage sub-issues

USAGE
  gh subissue <command> [flags]

COMMANDS
  create    Create a new sub-issue
  list      List sub-issues of a parent issue
  edit      Edit an existing sub-issue (add to project, etc.)
  repos     List repositories and their sub-issue availability

CREATE FLAGS
  -p, --parent <number>    Parent issue number (interactive if omitted)
  -t, --title <string>     Issue title
  -b, --body <string>      Issue body
      --body-file <file>   Read body from file (use - for stdin)
  -R, --repo <owner/repo>  Repository (defaults to current)
  -a, --assignee <user>    Assign users (can repeat)
  -l, --label <name>       Add labels (can repeat)
  -m, --milestone <number> Milestone number
  -P, --project <name>     Add to project (interactive if empty)
  -w, --web                Open in browser after creation

LIST FLAGS
  -p, --parent <number>    Parent issue number (interactive if omitted)
  -R, --repo <owner/repo>  Repository (defaults to current)

EDIT FLAGS
  <issue-number>           Issue number to edit (required)
  -P, --project <name>     Add to project (interactive if empty)
  -R, --repo <owner/repo>  Repository (defaults to current)

REPOS FLAGS
  [<owner>]                User or organization to list repos for (defaults to you)
  -L, --limit <int>        Maximum repos to list (default 30)
      --enabled            Show only repos where sub-issues work
      --disabled           Show only repos where sub-issues don't work

ENVIRONMENT VARIABLES
  GH_DEBUG                 Set to any value to enable debug logging (logfmt to stderr)

EXAMPLES
  gh subissue create --title "New task"                           # Interactive parent selection
  gh subissue create --parent 42 --title "Implement feature"
  gh subissue create -p 42 -t "Fix bug" -l bug -a username
  gh subissue create -p 42 -t "Task" --project "Roadmap"          # Add to specific project
  gh subissue list --parent 42                                    # List sub-issues
  gh subissue list                                                # Interactive parent selection
  gh subissue edit 43 --project "Roadmap"                         # Add issue to project
  gh subissue repos                                               # List your repos
  gh subissue repos my-org                                        # List org repos
  gh subissue repos --enabled                                     # Only repos with sub-issues
  GH_DEBUG=1 gh subissue create -p 42 -t "Debug me"               # Enable debug logging
`
	fmt.Print(usage)
	return nil
}

// Compile-time checks
var _ cmd.APIClient = (*internalapi.Client)(nil)
var _ cmd.ListAPIClient = (*internalapi.Client)(nil)
var _ cmd.EditAPIClient = (*internalapi.Client)(nil)
var _ cmd.ReposAPIClient = (*internalapi.Client)(nil)
