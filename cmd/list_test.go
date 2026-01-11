package cmd

import (
	"bytes"
	"testing"

	"github.com/gwyn/gh-subissue/internal/api"
)

func TestParseListFlags(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantParent int
		wantRepo   string
		wantErr    bool
	}{
		{
			name:       "with parent flag",
			args:       []string{"--parent", "42"},
			wantParent: 42,
			wantErr:    false,
		},
		{
			name:       "with short parent flag",
			args:       []string{"-p", "42"},
			wantParent: 42,
			wantErr:    false,
		},
		{
			name:       "with repo flag",
			args:       []string{"-p", "42", "--repo", "owner/repo"},
			wantParent: 42,
			wantRepo:   "owner/repo",
			wantErr:    false,
		},
		{
			name:       "missing parent flag",
			args:       []string{},
			wantParent: 0,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts, err := ParseListFlags(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseListFlags() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if opts.Parent != tt.wantParent {
				t.Errorf("Parent = %d, want %d", opts.Parent, tt.wantParent)
			}
			if opts.Repo != tt.wantRepo {
				t.Errorf("Repo = %q, want %q", opts.Repo, tt.wantRepo)
			}
		})
	}
}

// mockListAPIClient implements the ListAPIClient interface for testing.
type mockListAPIClient struct {
	listSubIssuesFunc func(opts api.ListSubIssuesOptions) ([]api.Issue, error)
	listIssuesFunc    func(opts api.ListIssuesOptions) ([]api.Issue, error)
}

func (m *mockListAPIClient) ListSubIssues(opts api.ListSubIssuesOptions) ([]api.Issue, error) {
	if m.listSubIssuesFunc != nil {
		return m.listSubIssuesFunc(opts)
	}
	return []api.Issue{}, nil
}

func (m *mockListAPIClient) ListIssues(opts api.ListIssuesOptions) ([]api.Issue, error) {
	if m.listIssuesFunc != nil {
		return m.listIssuesFunc(opts)
	}
	return []api.Issue{}, nil
}

// Compile-time check
var _ ListAPIClient = (*mockListAPIClient)(nil)

func TestListRunnerRun(t *testing.T) {
	tests := []struct {
		name       string
		opts       ListOptions
		subIssues  []api.Issue
		wantOutput string
		wantErr    bool
	}{
		{
			name: "lists sub-issues",
			opts: ListOptions{Parent: 42},
			subIssues: []api.Issue{
				{Number: 43, Title: "Sub-issue 1", URL: "https://github.com/owner/repo/issues/43"},
				{Number: 44, Title: "Sub-issue 2", URL: "https://github.com/owner/repo/issues/44"},
			},
			wantOutput: "#43\tSub-issue 1\n#44\tSub-issue 2\n",
			wantErr:    false,
		},
		{
			name:       "empty list",
			opts:       ListOptions{Parent: 42},
			subIssues:  []api.Issue{},
			wantOutput: "No sub-issues found for issue #42\n",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &mockListAPIClient{
				listSubIssuesFunc: func(opts api.ListSubIssuesOptions) ([]api.Issue, error) {
					return tt.subIssues, nil
				},
			}

			var output bytes.Buffer
			runner := &ListRunner{
				Client: client,
				Owner:  "owner",
				Repo:   "repo",
				Out:    &output,
			}

			err := runner.Run(tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if output.String() != tt.wantOutput {
				t.Errorf("output = %q, want %q", output.String(), tt.wantOutput)
			}
		})
	}
}

func TestListRunnerInteractiveParentSelection(t *testing.T) {
	client := &mockListAPIClient{
		listIssuesFunc: func(opts api.ListIssuesOptions) ([]api.Issue, error) {
			return []api.Issue{
				{Number: 10, Title: "Parent Issue 1"},
				{Number: 20, Title: "Parent Issue 2"},
			}, nil
		},
		listSubIssuesFunc: func(opts api.ListSubIssuesOptions) ([]api.Issue, error) {
			if opts.ParentIssue != 10 {
				t.Errorf("expected parent 10, got %d", opts.ParentIssue)
			}
			return []api.Issue{
				{Number: 43, Title: "Sub-issue 1"},
			}, nil
		},
	}

	prompter := &mockPrompter{
		selectFunc: func(prompt, defaultValue string, options []string) (int, error) {
			return 0, nil // Select first option
		},
	}

	var output bytes.Buffer
	runner := &ListRunner{
		Client:   client,
		Owner:    "owner",
		Repo:     "repo",
		Out:      &output,
		Prompter: prompter,
	}

	err := runner.Run(ListOptions{Parent: 0})
	if err != nil {
		t.Errorf("Run() error = %v", err)
	}
}
