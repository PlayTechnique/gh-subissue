package cmd

import (
	"bytes"
	"errors"
	"testing"

	"github.com/gwyn/gh-subissue/internal/api"
)

func TestParseReposFlags(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		wantOwner    string
		wantLimit    int
		wantEnabled  bool
		wantDisabled bool
		wantErr      bool
	}{
		{
			name:      "no arguments uses defaults",
			args:      []string{},
			wantOwner: "",
			wantLimit: 30,
			wantErr:   false,
		},
		{
			name:      "positional owner argument",
			args:      []string{"myorg"},
			wantOwner: "myorg",
			wantLimit: 30,
			wantErr:   false,
		},
		{
			name:      "limit flag short",
			args:      []string{"-L", "50"},
			wantLimit: 50,
			wantErr:   false,
		},
		{
			name:      "limit flag long",
			args:      []string{"--limit", "100"},
			wantLimit: 100,
			wantErr:   false,
		},
		{
			name:        "enabled filter",
			args:        []string{"--enabled"},
			wantEnabled: true,
			wantLimit:   30,
			wantErr:     false,
		},
		{
			name:         "disabled filter",
			args:         []string{"--disabled"},
			wantDisabled: true,
			wantLimit:    30,
			wantErr:      false,
		},
		{
			name:         "owner with filters",
			args:         []string{"myorg", "--enabled", "-L", "10"},
			wantOwner:    "myorg",
			wantEnabled:  true,
			wantLimit:    10,
			wantErr:      false,
		},
		{
			name:    "invalid limit",
			args:    []string{"-L", "notanumber"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts, err := ParseReposFlags(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseReposFlags() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if opts.Owner != tt.wantOwner {
				t.Errorf("Owner = %q, want %q", opts.Owner, tt.wantOwner)
			}
			if opts.Limit != tt.wantLimit {
				t.Errorf("Limit = %d, want %d", opts.Limit, tt.wantLimit)
			}
			if opts.Enabled != tt.wantEnabled {
				t.Errorf("Enabled = %v, want %v", opts.Enabled, tt.wantEnabled)
			}
			if opts.Disabled != tt.wantDisabled {
				t.Errorf("Disabled = %v, want %v", opts.Disabled, tt.wantDisabled)
			}
		})
	}
}

// mockReposAPIClient implements the ReposAPIClient interface for testing.
type mockReposAPIClient struct {
	listRepositoriesFunc   func(opts api.ListRepositoriesOptions) ([]api.Repository, error)
	getAuthenticatedUserFunc func() (*api.User, error)
}

func (m *mockReposAPIClient) ListRepositories(opts api.ListRepositoriesOptions) ([]api.Repository, error) {
	if m.listRepositoriesFunc != nil {
		return m.listRepositoriesFunc(opts)
	}
	return []api.Repository{}, nil
}

func (m *mockReposAPIClient) GetAuthenticatedUser() (*api.User, error) {
	if m.getAuthenticatedUserFunc != nil {
		return m.getAuthenticatedUserFunc()
	}
	return &api.User{Login: "testuser"}, nil
}

// Compile-time check
var _ ReposAPIClient = (*mockReposAPIClient)(nil)

func TestReposRunnerRun(t *testing.T) {
	tests := []struct {
		name       string
		opts       ReposOptions
		repos      []api.Repository
		wantOutput string
		wantErr    bool
	}{
		{
			name: "lists repos with status",
			opts: ReposOptions{Owner: "testorg", Limit: 30},
			repos: []api.Repository{
				{Name: "repo1", FullName: "testorg/repo1", HasIssues: true, Archived: false},
				{Name: "repo2", FullName: "testorg/repo2", HasIssues: false, Archived: false},
				{Name: "repo3", FullName: "testorg/repo3", HasIssues: true, Archived: true},
			},
			wantOutput: "REPOSITORY          SUB-ISSUES\ntestorg/repo1       enabled\ntestorg/repo2       disabled (issues off)\ntestorg/repo3       disabled (archived)\n",
			wantErr:    false,
		},
		{
			name:       "empty list",
			opts:       ReposOptions{Owner: "emptyorg", Limit: 30},
			repos:      []api.Repository{},
			wantOutput: "No repositories found for emptyorg\n",
			wantErr:    false,
		},
		{
			name: "enabled filter",
			opts: ReposOptions{Owner: "testorg", Limit: 30, Enabled: true},
			repos: []api.Repository{
				{Name: "repo1", FullName: "testorg/repo1", HasIssues: true, Archived: false},
				{Name: "repo2", FullName: "testorg/repo2", HasIssues: false, Archived: false},
				{Name: "repo3", FullName: "testorg/repo3", HasIssues: true, Archived: true},
			},
			wantOutput: "REPOSITORY          SUB-ISSUES\ntestorg/repo1       enabled\n",
			wantErr:    false,
		},
		{
			name: "disabled filter",
			opts: ReposOptions{Owner: "testorg", Limit: 30, Disabled: true},
			repos: []api.Repository{
				{Name: "repo1", FullName: "testorg/repo1", HasIssues: true, Archived: false},
				{Name: "repo2", FullName: "testorg/repo2", HasIssues: false, Archived: false},
				{Name: "repo3", FullName: "testorg/repo3", HasIssues: true, Archived: true},
			},
			wantOutput: "REPOSITORY          SUB-ISSUES\ntestorg/repo2       disabled (issues off)\ntestorg/repo3       disabled (archived)\n",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &mockReposAPIClient{
				listRepositoriesFunc: func(opts api.ListRepositoriesOptions) ([]api.Repository, error) {
					return tt.repos, nil
				},
			}

			var output bytes.Buffer
			runner := &ReposRunner{
				Client: client,
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

func TestReposRunnerRun_UsesAuthenticatedUser(t *testing.T) {
	client := &mockReposAPIClient{
		getAuthenticatedUserFunc: func() (*api.User, error) {
			return &api.User{Login: "myuser"}, nil
		},
		listRepositoriesFunc: func(opts api.ListRepositoriesOptions) ([]api.Repository, error) {
			if opts.Owner != "myuser" {
				t.Errorf("expected owner 'myuser', got %q", opts.Owner)
			}
			return []api.Repository{
				{Name: "myrepo", FullName: "myuser/myrepo", HasIssues: true},
			}, nil
		},
	}

	var output bytes.Buffer
	runner := &ReposRunner{
		Client: client,
		Out:    &output,
	}

	// No owner specified - should use authenticated user
	err := runner.Run(ReposOptions{Limit: 30})
	if err != nil {
		t.Errorf("Run() error = %v", err)
	}
}

func TestReposRunnerRun_APIError(t *testing.T) {
	client := &mockReposAPIClient{
		listRepositoriesFunc: func(opts api.ListRepositoriesOptions) ([]api.Repository, error) {
			return nil, errors.New("API error")
		},
	}

	var output bytes.Buffer
	runner := &ReposRunner{
		Client: client,
		Out:    &output,
	}

	err := runner.Run(ReposOptions{Owner: "testorg", Limit: 30})
	if err == nil {
		t.Error("Run() expected error, got nil")
	}
}

func TestReposRunnerRun_LimitRespected(t *testing.T) {
	repos := []api.Repository{
		{Name: "repo1", FullName: "org/repo1", HasIssues: true},
		{Name: "repo2", FullName: "org/repo2", HasIssues: true},
		{Name: "repo3", FullName: "org/repo3", HasIssues: true},
		{Name: "repo4", FullName: "org/repo4", HasIssues: true},
		{Name: "repo5", FullName: "org/repo5", HasIssues: true},
	}

	client := &mockReposAPIClient{
		listRepositoriesFunc: func(opts api.ListRepositoriesOptions) ([]api.Repository, error) {
			return repos, nil
		},
	}

	var output bytes.Buffer
	runner := &ReposRunner{
		Client: client,
		Out:    &output,
	}

	// Limit to 3 repos
	err := runner.Run(ReposOptions{Owner: "org", Limit: 3})
	if err != nil {
		t.Errorf("Run() error = %v", err)
	}

	// Count output lines (header + 3 repos = 4 lines)
	lines := bytes.Count(output.Bytes(), []byte("\n"))
	if lines != 4 {
		t.Errorf("expected 4 lines (header + 3 repos), got %d: %q", lines, output.String())
	}
}
