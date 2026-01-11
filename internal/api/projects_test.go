package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListProjects(t *testing.T) {
	tests := []struct {
		name         string
		response     string
		statusCode   int
		wantProjects int
		wantErr      bool
	}{
		{
			name: "returns projects from repository",
			response: `{
				"data": {
					"repository": {
						"projectsV2": {
							"nodes": [
								{"id": "PVT_1", "title": "Roadmap", "number": 1},
								{"id": "PVT_2", "title": "Sprint", "number": 2}
							]
						}
					}
				}
			}`,
			statusCode:   http.StatusOK,
			wantProjects: 2,
			wantErr:      false,
		},
		{
			name: "returns empty list when no projects",
			response: `{
				"data": {
					"repository": {
						"projectsV2": {
							"nodes": []
						}
					}
				}
			}`,
			statusCode:   http.StatusOK,
			wantProjects: 0,
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" {
					t.Errorf("expected POST, got %s", r.Method)
				}
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.response))
			}))
			defer server.Close()

			client := &Client{
				HTTPClient: server.Client(),
				BaseURL:    server.URL,
			}

			projects, err := client.ListProjects("owner", "repo")
			if (err != nil) != tt.wantErr {
				t.Errorf("ListProjects() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(projects) != tt.wantProjects {
				t.Errorf("got %d projects, want %d", len(projects), tt.wantProjects)
			}
		})
	}
}

func TestAddIssueToProject(t *testing.T) {
	tests := []struct {
		name       string
		response   string
		statusCode int
		wantErr    bool
	}{
		{
			name: "successfully adds issue to project",
			response: `{
				"data": {
					"addProjectV2ItemById": {
						"item": {
							"id": "PVTI_123"
						}
					}
				}
			}`,
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name: "returns error on GraphQL error",
			response: `{
				"errors": [
					{"message": "Project not found"}
				]
			}`,
			statusCode: http.StatusOK,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.response))
			}))
			defer server.Close()

			client := &Client{
				HTTPClient: server.Client(),
				BaseURL:    server.URL,
			}

			err := client.AddIssueToProject("PVT_123", "I_abc123")
			if (err != nil) != tt.wantErr {
				t.Errorf("AddIssueToProject() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetIssueNodeID(t *testing.T) {
	tests := []struct {
		name       string
		response   string
		statusCode int
		wantNodeID string
		wantErr    bool
	}{
		{
			name: "returns issue node ID",
			response: `{
				"data": {
					"repository": {
						"issue": {
							"id": "I_abc123"
						}
					}
				}
			}`,
			statusCode: http.StatusOK,
			wantNodeID: "I_abc123",
			wantErr:    false,
		},
		{
			name: "returns error when issue not found",
			response: `{
				"errors": [
					{"message": "Could not resolve to an Issue"}
				]
			}`,
			statusCode: http.StatusOK,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.response))
			}))
			defer server.Close()

			client := &Client{
				HTTPClient: server.Client(),
				BaseURL:    server.URL,
			}

			nodeID, err := client.GetIssueNodeID("owner", "repo", 42)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetIssueNodeID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && nodeID != tt.wantNodeID {
				t.Errorf("got nodeID %q, want %q", nodeID, tt.wantNodeID)
			}
		})
	}
}
