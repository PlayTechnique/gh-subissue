package cmd

import (
	"bytes"
	"testing"

	"github.com/gwyn/gh-subissue/internal/api"
)

func TestParseEditFlags(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantIssue   int
		wantProject OptionalString
		wantRepo    string
		wantErr     bool
	}{
		{
			name:        "issue number as first arg",
			args:        []string{"43"},
			wantIssue:   43,
			wantProject: OptionalString{WasSet: false},
			wantErr:     false,
		},
		{
			name:      "issue number with project",
			args:      []string{"43", "--project", "Roadmap"},
			wantIssue: 43,
			wantProject: OptionalString{
				Value:  "Roadmap",
				WasSet: true,
			},
			wantErr: false,
		},
		{
			name:      "issue number with empty project (interactive)",
			args:      []string{"43", "--project", ""},
			wantIssue: 43,
			wantProject: OptionalString{
				Value:  "",
				WasSet: true,
			},
			wantErr: false,
		},
		{
			name:      "issue number with repo",
			args:      []string{"43", "--repo", "owner/repo"},
			wantIssue: 43,
			wantRepo:  "owner/repo",
			wantErr:   false,
		},
		{
			name:    "no issue number",
			args:    []string{"--project", "Roadmap"},
			wantErr: true,
		},
		{
			name:    "invalid issue number",
			args:    []string{"abc"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts, err := ParseEditFlags(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseEditFlags() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if opts.IssueNumber != tt.wantIssue {
				t.Errorf("IssueNumber = %d, want %d", opts.IssueNumber, tt.wantIssue)
			}
			if opts.Project.WasSet != tt.wantProject.WasSet {
				t.Errorf("Project.WasSet = %v, want %v", opts.Project.WasSet, tt.wantProject.WasSet)
			}
			if opts.Project.Value != tt.wantProject.Value {
				t.Errorf("Project.Value = %q, want %q", opts.Project.Value, tt.wantProject.Value)
			}
			if opts.Repo != tt.wantRepo {
				t.Errorf("Repo = %q, want %q", opts.Repo, tt.wantRepo)
			}
		})
	}
}

// mockEditAPIClient implements the EditAPIClient interface for testing.
type mockEditAPIClient struct {
	listProjectsFunc      func(owner, repo string) ([]api.Project, error)
	getIssueNodeIDFunc    func(owner, repo string, number int) (string, error)
	addIssueToProjectFunc func(projectID, issueNodeID string) error
}

func (m *mockEditAPIClient) ListProjects(owner, repo string) ([]api.Project, error) {
	if m.listProjectsFunc != nil {
		return m.listProjectsFunc(owner, repo)
	}
	return []api.Project{}, nil
}

func (m *mockEditAPIClient) GetIssueNodeID(owner, repo string, number int) (string, error) {
	if m.getIssueNodeIDFunc != nil {
		return m.getIssueNodeIDFunc(owner, repo, number)
	}
	return "I_mock", nil
}

func (m *mockEditAPIClient) AddIssueToProject(projectID, issueNodeID string) error {
	if m.addIssueToProjectFunc != nil {
		return m.addIssueToProjectFunc(projectID, issueNodeID)
	}
	return nil
}

// Compile-time check
var _ EditAPIClient = (*mockEditAPIClient)(nil)

func TestEditRunnerAddToProject(t *testing.T) {
	addCalled := false
	client := &mockEditAPIClient{
		listProjectsFunc: func(owner, repo string) ([]api.Project, error) {
			return []api.Project{
				{ID: "PVT_1", Title: "Roadmap", Number: 1},
				{ID: "PVT_2", Title: "Sprint", Number: 2},
			}, nil
		},
		getIssueNodeIDFunc: func(owner, repo string, number int) (string, error) {
			if number != 43 {
				t.Errorf("expected issue 43, got %d", number)
			}
			return "I_abc123", nil
		},
		addIssueToProjectFunc: func(projectID, issueNodeID string) error {
			addCalled = true
			if projectID != "PVT_1" {
				t.Errorf("expected project PVT_1, got %s", projectID)
			}
			return nil
		},
	}

	var output bytes.Buffer
	runner := &EditRunner{
		Client: client,
		Owner:  "owner",
		Repo:   "repo",
		Out:    &output,
	}

	opts := EditOptions{
		IssueNumber: 43,
		Project: OptionalString{
			Value:  "Roadmap",
			WasSet: true,
		},
	}

	err := runner.Run(opts)
	if err != nil {
		t.Errorf("Run() error = %v", err)
	}
	if !addCalled {
		t.Error("AddIssueToProject was not called")
	}
}

func TestEditRunnerInteractiveProject(t *testing.T) {
	client := &mockEditAPIClient{
		listProjectsFunc: func(owner, repo string) ([]api.Project, error) {
			return []api.Project{
				{ID: "PVT_1", Title: "Roadmap", Number: 1},
				{ID: "PVT_2", Title: "Sprint", Number: 2},
			}, nil
		},
		getIssueNodeIDFunc: func(owner, repo string, number int) (string, error) {
			return "I_abc123", nil
		},
		addIssueToProjectFunc: func(projectID, issueNodeID string) error {
			if projectID != "PVT_2" {
				t.Errorf("expected project PVT_2 (second), got %s", projectID)
			}
			return nil
		},
	}

	prompter := &mockPrompter{
		selectFunc: func(prompt, defaultValue string, options []string) (int, error) {
			return 1, nil // Select second project
		},
	}

	var output bytes.Buffer
	runner := &EditRunner{
		Client:   client,
		Owner:    "owner",
		Repo:     "repo",
		Out:      &output,
		Prompter: prompter,
	}

	opts := EditOptions{
		IssueNumber: 43,
		Project: OptionalString{
			Value:  "", // Interactive mode
			WasSet: true,
		},
	}

	err := runner.Run(opts)
	if err != nil {
		t.Errorf("Run() error = %v", err)
	}
}

func TestEditRunnerNoProject(t *testing.T) {
	var output bytes.Buffer
	runner := &EditRunner{
		Client: &mockEditAPIClient{},
		Owner:  "owner",
		Repo:   "repo",
		Out:    &output,
	}

	opts := EditOptions{
		IssueNumber: 43,
		// Project not set
	}

	err := runner.Run(opts)
	if err == nil {
		t.Error("expected error when no project specified")
	}
}
