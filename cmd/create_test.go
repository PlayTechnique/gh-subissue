package cmd

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/gwyn/gh-subissue/internal/api"
)

// mockAPIClient implements the API interface for testing.
type mockAPIClient struct {
	createIssueFunc  func(opts api.CreateIssueOptions) (*api.IssueResult, error)
	linkSubIssueFunc func(opts api.LinkSubIssueOptions) error
	getIssueFunc     func(owner, repo string, number int) (*api.Issue, error)
	listIssuesFunc   func(opts api.ListIssuesOptions) ([]api.Issue, error)
}

func (m *mockAPIClient) CreateIssue(opts api.CreateIssueOptions) (*api.IssueResult, error) {
	if m.createIssueFunc != nil {
		return m.createIssueFunc(opts)
	}
	return &api.IssueResult{ID: 1, Number: 1, URL: "https://github.com/test/test/issues/1"}, nil
}

func (m *mockAPIClient) LinkSubIssue(opts api.LinkSubIssueOptions) error {
	if m.linkSubIssueFunc != nil {
		return m.linkSubIssueFunc(opts)
	}
	return nil
}

func (m *mockAPIClient) GetIssue(owner, repo string, number int) (*api.Issue, error) {
	if m.getIssueFunc != nil {
		return m.getIssueFunc(owner, repo, number)
	}
	return &api.Issue{ID: 99999, Number: number, Title: "Parent"}, nil
}

func (m *mockAPIClient) ListIssues(opts api.ListIssuesOptions) ([]api.Issue, error) {
	if m.listIssuesFunc != nil {
		return m.listIssuesFunc(opts)
	}
	return []api.Issue{}, nil
}

func TestParseFlags(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    Options
		wantErr bool
	}{
		{
			name: "all flags provided",
			args: []string{
				"--parent", "42",
				"--title", "Test Issue",
				"--body", "Test body",
				"--repo", "owner/repo",
				"--assignee", "user1",
				"--assignee", "user2",
				"--label", "bug",
				"--label", "priority",
				"--milestone", "5",
			},
			want: Options{
				Parent:    42,
				Title:     "Test Issue",
				Body:      "Test body",
				Repo:      "owner/repo",
				Assignees: []string{"user1", "user2"},
				Labels:    []string{"bug", "priority"},
				Milestone: 5,
			},
			wantErr: false,
		},
		{
			name: "short flags",
			args: []string{
				"-p", "42",
				"-t", "Test Issue",
				"-b", "Test body",
				"-R", "owner/repo",
				"-a", "user1",
				"-l", "bug",
				"-m", "5",
			},
			want: Options{
				Parent:    42,
				Title:     "Test Issue",
				Body:      "Test body",
				Repo:      "owner/repo",
				Assignees: []string{"user1"},
				Labels:    []string{"bug"},
				Milestone: 5,
			},
			wantErr: false,
		},
		{
			name: "no parent flag returns zero",
			args: []string{"--title", "Test Issue"},
			want: Options{
				Parent: 0,
				Title:  "Test Issue",
			},
			wantErr: false,
		},
		{
			name: "minimal flags",
			args: []string{
				"--parent", "42",
				"--title", "Minimal Issue",
			},
			want: Options{
				Parent: 42,
				Title:  "Minimal Issue",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts, err := ParseFlags(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseFlags() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			if opts.Parent != tt.want.Parent {
				t.Errorf("Parent = %v, want %v", opts.Parent, tt.want.Parent)
			}
			if opts.Title != tt.want.Title {
				t.Errorf("Title = %v, want %v", opts.Title, tt.want.Title)
			}
			if opts.Body != tt.want.Body {
				t.Errorf("Body = %v, want %v", opts.Body, tt.want.Body)
			}
			if opts.Repo != tt.want.Repo {
				t.Errorf("Repo = %v, want %v", opts.Repo, tt.want.Repo)
			}
			if opts.Milestone != tt.want.Milestone {
				t.Errorf("Milestone = %v, want %v", opts.Milestone, tt.want.Milestone)
			}
			if len(opts.Assignees) != len(tt.want.Assignees) {
				t.Errorf("Assignees len = %v, want %v", len(opts.Assignees), len(tt.want.Assignees))
			}
			if len(opts.Labels) != len(tt.want.Labels) {
				t.Errorf("Labels len = %v, want %v", len(opts.Labels), len(tt.want.Labels))
			}
		})
	}
}

func TestRun(t *testing.T) {
	tests := []struct {
		name       string
		opts       Options
		client     *mockAPIClient
		wantOutput string
		wantErr    bool
	}{
		{
			name: "successful create and link",
			opts: Options{
				Parent: 42,
				Title:  "Sub Issue",
				Body:   "Body text",
			},
			client: &mockAPIClient{
				createIssueFunc: func(opts api.CreateIssueOptions) (*api.IssueResult, error) {
					return &api.IssueResult{
						ID:     12345,
						Number: 43,
						URL:    "https://github.com/owner/repo/issues/43",
					}, nil
				},
				linkSubIssueFunc: func(opts api.LinkSubIssueOptions) error {
					if opts.ParentIssue != 42 {
						t.Errorf("expected parent 42, got %d", opts.ParentIssue)
					}
					if opts.SubIssueID != 12345 {
						t.Errorf("expected sub_issue_id 12345, got %d", opts.SubIssueID)
					}
					return nil
				},
			},
			wantOutput: "https://github.com/owner/repo/issues/43",
			wantErr:    false,
		},
		{
			name: "issue created but link fails - shows warning",
			opts: Options{
				Parent: 42,
				Title:  "Sub Issue",
			},
			client: &mockAPIClient{
				createIssueFunc: func(opts api.CreateIssueOptions) (*api.IssueResult, error) {
					return &api.IssueResult{
						ID:     12345,
						Number: 43,
						URL:    "https://github.com/owner/repo/issues/43",
					}, nil
				},
				linkSubIssueFunc: func(opts api.LinkSubIssueOptions) error {
					return errors.New("permission denied")
				},
			},
			wantOutput: "Warning:",
			wantErr:    false, // Should not error, just warn
		},
		{
			name: "issue creation fails",
			opts: Options{
				Parent: 42,
				Title:  "Sub Issue",
			},
			client: &mockAPIClient{
				createIssueFunc: func(opts api.CreateIssueOptions) (*api.IssueResult, error) {
					return nil, errors.New("validation failed")
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			runner := &Runner{
				Client: tt.client,
				Owner:  "owner",
				Repo:   "repo",
				Out:    &output,
			}

			err := runner.Run(tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantOutput != "" && !strings.Contains(output.String(), tt.wantOutput) {
				t.Errorf("output = %q, want to contain %q", output.String(), tt.wantOutput)
			}
		})
	}
}

func TestReadBodyFromStdin(t *testing.T) {
	input := "This is the body from stdin"
	reader := strings.NewReader(input)

	body, err := ReadBody("-", reader)
	if err != nil {
		t.Errorf("ReadBody() error = %v", err)
	}
	if body != input {
		t.Errorf("ReadBody() = %q, want %q", body, input)
	}
}

func TestReadBodyFromFile(t *testing.T) {
	// Create a temp file reader simulation
	content := "Body from file"
	reader := strings.NewReader(content)

	// When path is "-", read from the provided reader
	body, err := ReadBody("-", reader)
	if err != nil {
		t.Errorf("ReadBody() error = %v", err)
	}
	if body != content {
		t.Errorf("ReadBody() = %q, want %q", body, content)
	}
}

// Verify mockAPIClient implements APIClient
var _ APIClient = (*mockAPIClient)(nil)

func TestParseRepo(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantOwner string
		wantRepo  string
		wantErr   bool
	}{
		{
			name:      "valid owner/repo",
			input:     "owner/repo",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantErr:   false,
		},
		{
			name:    "invalid format - no slash",
			input:   "ownerrepo",
			wantErr: true,
		},
		{
			name:    "invalid format - too many slashes",
			input:   "owner/repo/extra",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, err := ParseRepo(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseRepo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if owner != tt.wantOwner {
					t.Errorf("owner = %v, want %v", owner, tt.wantOwner)
				}
				if repo != tt.wantRepo {
					t.Errorf("repo = %v, want %v", repo, tt.wantRepo)
				}
			}
		})
	}
}

// Test that web flag is parsed
func TestParseFlagsWebFlag(t *testing.T) {
	args := []string{
		"--parent", "42",
		"--title", "Test",
		"--web",
	}

	opts, err := ParseFlags(args)
	if err != nil {
		t.Errorf("ParseFlags() error = %v", err)
	}
	if !opts.Web {
		t.Error("Web flag should be true")
	}
}

// Test body-file flag
func TestParseFlagsBodyFile(t *testing.T) {
	args := []string{
		"--parent", "42",
		"--title", "Test",
		"--body-file", "-",
	}

	opts, err := ParseFlags(args)
	if err != nil {
		t.Errorf("ParseFlags() error = %v", err)
	}
	if opts.BodyFile != "-" {
		t.Errorf("BodyFile = %q, want %q", opts.BodyFile, "-")
	}
}

// Ensure body and body-file are properly handled
func TestRunWithBodyFile(t *testing.T) {
	client := &mockAPIClient{
		createIssueFunc: func(opts api.CreateIssueOptions) (*api.IssueResult, error) {
			if opts.Body != "Body from stdin" {
				t.Errorf("Body = %q, want %q", opts.Body, "Body from stdin")
			}
			return &api.IssueResult{ID: 1, Number: 1, URL: "https://github.com/o/r/issues/1"}, nil
		},
	}

	var output bytes.Buffer
	runner := &Runner{
		Client: client,
		Owner:  "owner",
		Repo:   "repo",
		Out:    &output,
		Stdin:  strings.NewReader("Body from stdin"),
	}

	opts := Options{
		Parent:   42,
		Title:    "Test",
		BodyFile: "-",
	}

	err := runner.Run(opts)
	if err != nil {
		t.Errorf("Run() error = %v", err)
	}
}

// TestRunner with validation of parent existence
func TestRunValidatesParentExists(t *testing.T) {
	client := &mockAPIClient{
		getIssueFunc: func(owner, repo string, number int) (*api.Issue, error) {
			return nil, errors.New("not found")
		},
	}

	var output bytes.Buffer
	runner := &Runner{
		Client:         client,
		Owner:          "owner",
		Repo:           "repo",
		Out:            &output,
		ValidateParent: true,
	}

	opts := Options{
		Parent: 999,
		Title:  "Test",
	}

	err := runner.Run(opts)
	if err == nil {
		t.Error("expected error when parent doesn't exist")
	}
	if !strings.Contains(err.Error(), "parent") {
		t.Errorf("error should mention parent: %v", err)
	}
}

// Ensure ReadBody can read from an io.Reader
func TestReadBodyReader(t *testing.T) {
	content := "test content"
	r := strings.NewReader(content)

	body, err := ReadBody("-", r)
	if err != nil {
		t.Fatalf("ReadBody error: %v", err)
	}
	if body != content {
		t.Errorf("got %q, want %q", body, content)
	}
}

// Test that we need a non-nil reader for stdin
func TestReadBodyNilReader(t *testing.T) {
	_, err := ReadBody("-", nil)
	if err == nil {
		t.Error("expected error with nil reader")
	}
}

// Ensure io interface is used
var _ io.Writer = (*bytes.Buffer)(nil)
var _ io.Reader = (*strings.Reader)(nil)

// mockPrompterInCreate implements Prompter for testing.
type mockPrompterInCreate struct {
	selectFunc func(prompt string, defaultValue string, options []string) (int, error)
	inputFunc  func(prompt, defaultValue string) (string, error)
}

func (m *mockPrompterInCreate) Select(prompt, defaultValue string, options []string) (int, error) {
	if m.selectFunc != nil {
		return m.selectFunc(prompt, defaultValue, options)
	}
	return 0, nil
}

func (m *mockPrompterInCreate) Input(prompt, defaultValue string) (string, error) {
	if m.inputFunc != nil {
		return m.inputFunc(prompt, defaultValue)
	}
	return "", nil
}

var _ Prompter = (*mockPrompterInCreate)(nil)

func TestRunInteractiveSelection(t *testing.T) {
	client := &mockAPIClient{
		listIssuesFunc: func(opts api.ListIssuesOptions) ([]api.Issue, error) {
			return []api.Issue{
				{ID: 100, Number: 10, Title: "Parent Issue"},
				{ID: 200, Number: 20, Title: "Another Issue"},
			}, nil
		},
		createIssueFunc: func(opts api.CreateIssueOptions) (*api.IssueResult, error) {
			return &api.IssueResult{
				ID:     300,
				Number: 30,
				URL:    "https://github.com/owner/repo/issues/30",
			}, nil
		},
		linkSubIssueFunc: func(opts api.LinkSubIssueOptions) error {
			if opts.ParentIssue != 10 {
				t.Errorf("expected parent 10, got %d", opts.ParentIssue)
			}
			return nil
		},
	}

	prompter := &mockPrompterInCreate{
		selectFunc: func(prompt, defaultValue string, options []string) (int, error) {
			// Select the first issue
			return 0, nil
		},
	}

	var output bytes.Buffer
	runner := &Runner{
		Client:   client,
		Owner:    "owner",
		Repo:     "repo",
		Out:      &output,
		Prompter: prompter,
	}

	opts := Options{
		Parent: 0, // No parent - should trigger interactive mode
		Title:  "Sub Issue",
	}

	err := runner.Run(opts)
	if err != nil {
		t.Errorf("Run() error = %v", err)
	}

	if !strings.Contains(output.String(), "issues/30") {
		t.Errorf("expected output to contain issue URL, got %q", output.String())
	}
}

func TestRunNoPrompterRequiresParent(t *testing.T) {
	client := &mockAPIClient{}

	var output bytes.Buffer
	runner := &Runner{
		Client:   client,
		Owner:    "owner",
		Repo:     "repo",
		Out:      &output,
		Prompter: nil, // No prompter - non-interactive mode
	}

	opts := Options{
		Parent: 0, // No parent
		Title:  "Sub Issue",
	}

	err := runner.Run(opts)
	if err == nil {
		t.Error("expected error when parent=0 and no prompter")
	}
	if !strings.Contains(err.Error(), "--parent flag is required") {
		t.Errorf("error should mention --parent flag required, got: %v", err)
	}
	if !strings.Contains(err.Error(), "not running interactively") {
		t.Errorf("error should mention not running interactively, got: %v", err)
	}
	if !strings.Contains(err.Error(), "Tip:") {
		t.Errorf("error should include a tip, got: %v", err)
	}
}

func TestRunListIssuesError(t *testing.T) {
	client := &mockAPIClient{
		listIssuesFunc: func(opts api.ListIssuesOptions) ([]api.Issue, error) {
			return nil, errors.New("API error")
		},
	}

	prompter := &mockPrompterInCreate{
		selectFunc: func(prompt, defaultValue string, options []string) (int, error) {
			return 0, nil
		},
	}

	var output bytes.Buffer
	runner := &Runner{
		Client:   client,
		Owner:    "owner",
		Repo:     "repo",
		Out:      &output,
		Prompter: prompter,
	}

	opts := Options{
		Parent: 0,
		Title:  "Sub Issue",
	}

	err := runner.Run(opts)
	if err == nil {
		t.Error("expected error when ListIssues fails")
	}
	if !strings.Contains(err.Error(), "list issues") {
		t.Errorf("error should mention list issues, got: %v", err)
	}
}

func TestRunInteractiveTitlePrompt(t *testing.T) {
	client := &mockAPIClient{
		createIssueFunc: func(opts api.CreateIssueOptions) (*api.IssueResult, error) {
			if opts.Title != "Prompted Title" {
				t.Errorf("expected title 'Prompted Title', got %q", opts.Title)
			}
			return &api.IssueResult{
				ID:     300,
				Number: 30,
				URL:    "https://github.com/owner/repo/issues/30",
			}, nil
		},
		linkSubIssueFunc: func(opts api.LinkSubIssueOptions) error {
			return nil
		},
	}

	prompter := &mockPrompterInCreate{
		inputFunc: func(prompt, defaultValue string) (string, error) {
			if !strings.Contains(prompt, "Title") {
				t.Errorf("expected prompt to mention Title, got %q", prompt)
			}
			return "Prompted Title", nil
		},
	}

	var output bytes.Buffer
	runner := &Runner{
		Client:   client,
		Owner:    "owner",
		Repo:     "repo",
		Out:      &output,
		Prompter: prompter,
	}

	opts := Options{
		Parent: 42,
		Title:  "", // No title - should trigger interactive prompt
	}

	err := runner.Run(opts)
	if err != nil {
		t.Errorf("Run() error = %v", err)
	}
}

func TestRunNoPrompterRequiresTitle(t *testing.T) {
	client := &mockAPIClient{}

	var output bytes.Buffer
	runner := &Runner{
		Client:   client,
		Owner:    "owner",
		Repo:     "repo",
		Out:      &output,
		Prompter: nil, // No prompter - non-interactive mode
	}

	opts := Options{
		Parent: 42,
		Title:  "", // No title
	}

	err := runner.Run(opts)
	if err == nil {
		t.Error("expected error when title is empty and no prompter")
	}
	if !strings.Contains(err.Error(), "--title") {
		t.Errorf("error should mention --title flag, got: %v", err)
	}
}

func TestRunInteractiveTitleCannotBeEmpty(t *testing.T) {
	prompter := &mockPrompterInCreate{
		inputFunc: func(prompt, defaultValue string) (string, error) {
			return "", nil // User enters empty string
		},
	}

	var output bytes.Buffer
	runner := &Runner{
		Client:   &mockAPIClient{},
		Owner:    "owner",
		Repo:     "repo",
		Out:      &output,
		Prompter: prompter,
	}

	opts := Options{
		Parent: 42,
		Title:  "",
	}

	err := runner.Run(opts)
	if err == nil {
		t.Error("expected error when user enters empty title")
	}
	if !strings.Contains(err.Error(), "title") {
		t.Errorf("error should mention title, got: %v", err)
	}
}
