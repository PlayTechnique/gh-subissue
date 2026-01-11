package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListRepositories(t *testing.T) {
	tests := []struct {
		name           string
		opts           ListRepositoriesOptions
		serverResponse func(w http.ResponseWriter, r *http.Request)
		wantCount      int
		wantErr        bool
	}{
		{
			name: "lists user repositories",
			opts: ListRepositoriesOptions{
				Owner:   "testuser",
				PerPage: 30,
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" {
					t.Errorf("expected GET, got %s", r.Method)
				}
				// Should try orgs first, then fall back to users
				if r.URL.Path == "/orgs/testuser/repos" {
					w.WriteHeader(http.StatusNotFound)
					json.NewEncoder(w).Encode(map[string]interface{}{
						"message": "Not Found",
					})
					return
				}
				if r.URL.Path != "/users/testuser/repos" {
					t.Errorf("unexpected path: %s", r.URL.Path)
				}
				if r.URL.Query().Get("per_page") != "30" {
					t.Errorf("expected per_page=30, got %s", r.URL.Query().Get("per_page"))
				}

				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode([]map[string]interface{}{
					{"name": "repo1", "full_name": "testuser/repo1", "has_issues": true, "archived": false, "private": false},
					{"name": "repo2", "full_name": "testuser/repo2", "has_issues": false, "archived": false, "private": true},
				})
			},
			wantCount: 2,
			wantErr:   false,
		},
		{
			name: "lists organization repositories",
			opts: ListRepositoriesOptions{
				Owner:   "testorg",
				PerPage: 30,
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" {
					t.Errorf("expected GET, got %s", r.Method)
				}
				if r.URL.Path != "/orgs/testorg/repos" {
					t.Errorf("unexpected path: %s", r.URL.Path)
				}

				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode([]map[string]interface{}{
					{"name": "org-repo1", "full_name": "testorg/org-repo1", "has_issues": true, "archived": false, "private": false},
					{"name": "org-repo2", "full_name": "testorg/org-repo2", "has_issues": true, "archived": true, "private": false},
				})
			},
			wantCount: 2,
			wantErr:   false,
		},
		{
			name: "empty repository list",
			opts: ListRepositoriesOptions{
				Owner:   "emptyuser",
				PerPage: 30,
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/orgs/emptyuser/repos" {
					w.WriteHeader(http.StatusNotFound)
					json.NewEncoder(w).Encode(map[string]interface{}{
						"message": "Not Found",
					})
					return
				}
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode([]map[string]interface{}{})
			},
			wantCount: 0,
			wantErr:   false,
		},
		{
			name: "handles not found for both org and user",
			opts: ListRepositoriesOptions{
				Owner:   "nonexistent",
				PerPage: 30,
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"message": "Not Found",
				})
			},
			wantErr: true,
		},
		{
			name: "respects page parameter",
			opts: ListRepositoriesOptions{
				Owner:   "testorg",
				PerPage: 10,
				Page:    2,
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Query().Get("page") != "2" {
					t.Errorf("expected page=2, got %s", r.URL.Query().Get("page"))
				}
				if r.URL.Query().Get("per_page") != "10" {
					t.Errorf("expected per_page=10, got %s", r.URL.Query().Get("per_page"))
				}

				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode([]map[string]interface{}{
					{"name": "repo11", "full_name": "testorg/repo11", "has_issues": true, "archived": false, "private": false},
				})
			},
			wantCount: 1,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.serverResponse))
			defer server.Close()

			client := &Client{
				HTTPClient: server.Client(),
				BaseURL:    server.URL,
			}

			repos, err := client.ListRepositories(tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("ListRepositories() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && len(repos) != tt.wantCount {
				t.Errorf("ListRepositories() returned %d repos, want %d", len(repos), tt.wantCount)
			}
		})
	}
}

func TestListRepositories_FieldsParsedCorrectly(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]map[string]interface{}{
			{
				"name":       "test-repo",
				"full_name":  "owner/test-repo",
				"has_issues": true,
				"archived":   false,
				"private":    true,
			},
		})
	}))
	defer server.Close()

	client := &Client{
		HTTPClient: server.Client(),
		BaseURL:    server.URL,
	}

	repos, err := client.ListRepositories(ListRepositoriesOptions{
		Owner:   "owner",
		PerPage: 30,
	})
	if err != nil {
		t.Fatalf("ListRepositories() error = %v", err)
	}

	if len(repos) != 1 {
		t.Fatalf("expected 1 repo, got %d", len(repos))
	}

	repo := repos[0]
	if repo.Name != "test-repo" {
		t.Errorf("Name = %q, want %q", repo.Name, "test-repo")
	}
	if repo.FullName != "owner/test-repo" {
		t.Errorf("FullName = %q, want %q", repo.FullName, "owner/test-repo")
	}
	if !repo.HasIssues {
		t.Error("HasIssues = false, want true")
	}
	if repo.Archived {
		t.Error("Archived = true, want false")
	}
	if !repo.Private {
		t.Error("Private = false, want true")
	}
}

func TestGetAuthenticatedUser(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse func(w http.ResponseWriter, r *http.Request)
		wantLogin      string
		wantErr        bool
	}{
		{
			name: "gets authenticated user",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" {
					t.Errorf("expected GET, got %s", r.Method)
				}
				if r.URL.Path != "/user" {
					t.Errorf("unexpected path: %s", r.URL.Path)
				}

				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"login": "currentuser",
				})
			},
			wantLogin: "currentuser",
			wantErr:   false,
		},
		{
			name: "handles unauthenticated",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"message": "Requires authentication",
				})
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.serverResponse))
			defer server.Close()

			client := &Client{
				HTTPClient: server.Client(),
				BaseURL:    server.URL,
			}

			user, err := client.GetAuthenticatedUser()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetAuthenticatedUser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && user.Login != tt.wantLogin {
				t.Errorf("GetAuthenticatedUser() login = %q, want %q", user.Login, tt.wantLogin)
			}
		})
	}
}
