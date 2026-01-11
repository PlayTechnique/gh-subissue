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
		repo, err := repository.Current()
		if err != nil {
			debug.Error("runCreate", err, "stage", "repository.Current")
			return fmt.Errorf("could not determine repository: %w (use --repo to specify)", err)
		}
		owner = repo.Owner
		repoName = repo.Name
		host = repo.Host
		debug.Log("runCreate", "resolved_repo", owner+"/"+repoName, "host", host)
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

	// Set up prompter for interactive mode
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

func printUsage() error {
	usage := `gh-subissue - Create sub-issues in a single command

USAGE
  gh subissue create [flags]

FLAGS
  -p, --parent <number>    Parent issue number (interactive if omitted)
  -t, --title <string>     Issue title
  -b, --body <string>      Issue body
      --body-file <file>   Read body from file (use - for stdin)
  -R, --repo <owner/repo>  Repository (defaults to current)
  -a, --assignee <user>    Assign users (can repeat)
  -l, --label <name>       Add labels (can repeat)
  -m, --milestone <number> Milestone number
  -w, --web                Open in browser after creation

ENVIRONMENT VARIABLES
  GH_DEBUG                 Set to any value to enable debug logging (logfmt to stderr)

EXAMPLES
  gh subissue create --title "New task"                           # Interactive parent selection
  gh subissue create --parent 42 --title "Implement feature"
  gh subissue create -p 42 -t "Fix bug" -l bug -a username
  echo "Details" | gh subissue create -p 42 -t "Task" --body-file -
  GH_DEBUG=1 gh subissue create -p 42 -t "Debug me"               # Enable debug logging
`
	fmt.Print(usage)
	return nil
}

// Compile-time check: internal/api.Client must implement cmd.APIClient
var _ cmd.APIClient = (*internalapi.Client)(nil)
