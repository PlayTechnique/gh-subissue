package main

import (
	"fmt"
	"os"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/browser"
	"github.com/cli/go-gh/v2/pkg/repository"

	"github.com/gwyn/gh-subissue/cmd"
	internalapi "github.com/gwyn/gh-subissue/internal/api"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	args := os.Args[1:]

	if len(args) == 0 {
		return printUsage()
	}

	switch args[0] {
	case "create":
		return runCreate(args[1:])
	case "help", "--help", "-h":
		return printUsage()
	case "version", "--version":
		fmt.Println("gh-subissue version 0.1.0")
		return nil
	default:
		return fmt.Errorf("unknown command: %s", args[0])
	}
}

func runCreate(args []string) error {
	opts, err := cmd.ParseFlags(args)
	if err != nil {
		return err
	}

	// Resolve repository
	var owner, repoName, host string
	if opts.Repo != "" {
		owner, repoName, err = cmd.ParseRepo(opts.Repo)
		if err != nil {
			return err
		}
		// When using --repo flag, try to detect host from current repo, fallback to github.com
		if repo, err := repository.Current(); err == nil {
			host = repo.Host
		} else {
			host = "github.com"
		}
	} else {
		repo, err := repository.Current()
		if err != nil {
			return fmt.Errorf("could not determine repository: %w (use --repo to specify)", err)
		}
		owner = repo.Owner
		repoName = repo.Name
		host = repo.Host
	}

	// Determine API base URL based on host
	var baseURL string
	if host == "github.com" || host == "" {
		baseURL = "https://api.github.com"
	} else {
		baseURL = fmt.Sprintf("https://%s/api/v3", host)
	}

	// Create authenticated HTTP client from go-gh
	httpClient, err := api.DefaultHTTPClient()
	if err != nil {
		return fmt.Errorf("failed to create HTTP client: %w", err)
	}

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
	}

	return runner.Run(*opts)
}

func printUsage() error {
	usage := `gh-subissue - Create sub-issues in a single command

USAGE
  gh subissue create [flags]

FLAGS
  -p, --parent <number>    Parent issue number (required)
  -t, --title <string>     Issue title
  -b, --body <string>      Issue body
      --body-file <file>   Read body from file (use - for stdin)
  -R, --repo <owner/repo>  Repository (defaults to current)
  -a, --assignee <user>    Assign users (can repeat)
  -l, --label <name>       Add labels (can repeat)
  -m, --milestone <number> Milestone number
  -w, --web                Open in browser after creation

EXAMPLES
  gh subissue create --parent 42 --title "Implement feature"
  gh subissue create -p 42 -t "Fix bug" -l bug -a username
  echo "Details" | gh subissue create -p 42 -t "Task" --body-file -
`
	fmt.Print(usage)
	return nil
}

// Compile-time check: internal/api.Client must implement cmd.APIClient
var _ cmd.APIClient = (*internalapi.Client)(nil)
