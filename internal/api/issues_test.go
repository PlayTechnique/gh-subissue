package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateIssue(t *testing.T) {
	tests := []struct {
		name           string
		opts           CreateIssueOptions
		serverResponse func(w http.ResponseWriter, r *http.Request)
		wantID         int64
		wantNumber     int
		wantURL        string
		wantErr        bool
	}{
		{
			name: "creates issue with title and body",
			opts: CreateIssueOptions{
				Owner: "testowner",
				Repo:  "testrepo",
				Title: "Test Issue",
				Body:  "Test body",
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" {
					t.Errorf("expected POST, got %s", r.Method)
				}
				if r.URL.Path != "/repos/testowner/testrepo/issues" {
					t.Errorf("unexpected path: %s", r.URL.Path)
				}

				var body map[string]interface{}
				json.NewDecoder(r.Body).Decode(&body)
				if body["title"] != "Test Issue" {
					t.Errorf("expected title 'Test Issue', got %v", body["title"])
				}
				if body["body"] != "Test body" {
					t.Errorf("expected body 'Test body', got %v", body["body"])
				}

				w.WriteHeader(http.StatusCreated)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"id":       12345,
					"number":   42,
					"html_url": "https://github.com/testowner/testrepo/issues/42",
				})
			},
			wantID:     12345,
			wantNumber: 42,
			wantURL:    "https://github.com/testowner/testrepo/issues/42",
			wantErr:    false,
		},
		{
			name: "creates issue with labels and assignees",
			opts: CreateIssueOptions{
				Owner:     "testowner",
				Repo:      "testrepo",
				Title:     "Labeled Issue",
				Labels:    []string{"bug", "priority"},
				Assignees: []string{"user1", "user2"},
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				var body map[string]interface{}
				json.NewDecoder(r.Body).Decode(&body)

				labels := body["labels"].([]interface{})
				if len(labels) != 2 {
					t.Errorf("expected 2 labels, got %d", len(labels))
				}

				assignees := body["assignees"].([]interface{})
				if len(assignees) != 2 {
					t.Errorf("expected 2 assignees, got %d", len(assignees))
				}

				w.WriteHeader(http.StatusCreated)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"id":       12346,
					"number":   43,
					"html_url": "https://github.com/testowner/testrepo/issues/43",
				})
			},
			wantID:     12346,
			wantNumber: 43,
			wantURL:    "https://github.com/testowner/testrepo/issues/43",
			wantErr:    false,
		},
		{
			name: "creates issue with milestone",
			opts: CreateIssueOptions{
				Owner:     "testowner",
				Repo:      "testrepo",
				Title:     "Milestone Issue",
				Milestone: 5,
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				var body map[string]interface{}
				json.NewDecoder(r.Body).Decode(&body)

				if body["milestone"].(float64) != 5 {
					t.Errorf("expected milestone 5, got %v", body["milestone"])
				}

				w.WriteHeader(http.StatusCreated)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"id":       12347,
					"number":   44,
					"html_url": "https://github.com/testowner/testrepo/issues/44",
				})
			},
			wantID:     12347,
			wantNumber: 44,
			wantURL:    "https://github.com/testowner/testrepo/issues/44",
			wantErr:    false,
		},
		{
			name: "handles server error",
			opts: CreateIssueOptions{
				Owner: "testowner",
				Repo:  "testrepo",
				Title: "Error Issue",
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnprocessableEntity)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"message": "Validation Failed",
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

			result, err := client.CreateIssue(tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateIssue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if result.ID != tt.wantID {
					t.Errorf("CreateIssue() ID = %v, want %v", result.ID, tt.wantID)
				}
				if result.Number != tt.wantNumber {
					t.Errorf("CreateIssue() Number = %v, want %v", result.Number, tt.wantNumber)
				}
				if result.URL != tt.wantURL {
					t.Errorf("CreateIssue() URL = %v, want %v", result.URL, tt.wantURL)
				}
			}
		})
	}
}

func TestLinkSubIssue(t *testing.T) {
	tests := []struct {
		name           string
		opts           LinkSubIssueOptions
		serverResponse func(w http.ResponseWriter, r *http.Request)
		wantErr        bool
	}{
		{
			name: "links sub-issue to parent",
			opts: LinkSubIssueOptions{
				Owner:       "testowner",
				Repo:        "testrepo",
				ParentIssue: 42,
				SubIssueID:  12345,
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" {
					t.Errorf("expected POST, got %s", r.Method)
				}
				if r.URL.Path != "/repos/testowner/testrepo/issues/42/sub_issues" {
					t.Errorf("unexpected path: %s", r.URL.Path)
				}

				var body map[string]interface{}
				json.NewDecoder(r.Body).Decode(&body)
				if body["sub_issue_id"].(float64) != 12345 {
					t.Errorf("expected sub_issue_id 12345, got %v", body["sub_issue_id"])
				}

				w.WriteHeader(http.StatusCreated)
				json.NewEncoder(w).Encode(map[string]interface{}{})
			},
			wantErr: false,
		},
		{
			name: "handles parent not found",
			opts: LinkSubIssueOptions{
				Owner:       "testowner",
				Repo:        "testrepo",
				ParentIssue: 999,
				SubIssueID:  12345,
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
			name: "handles permission error",
			opts: LinkSubIssueOptions{
				Owner:       "testowner",
				Repo:        "testrepo",
				ParentIssue: 42,
				SubIssueID:  12345,
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"message": "Must have write access",
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

			err := client.LinkSubIssue(tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("LinkSubIssue() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestListIssues(t *testing.T) {
	tests := []struct {
		name           string
		opts           ListIssuesOptions
		serverResponse func(w http.ResponseWriter, r *http.Request)
		wantCount      int
		wantErr        bool
	}{
		{
			name: "lists open issues",
			opts: ListIssuesOptions{
				Owner:   "testowner",
				Repo:    "testrepo",
				State:   "open",
				PerPage: 30,
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" {
					t.Errorf("expected GET, got %s", r.Method)
				}
				if r.URL.Path != "/repos/testowner/testrepo/issues" {
					t.Errorf("unexpected path: %s", r.URL.Path)
				}
				if r.URL.Query().Get("state") != "open" {
					t.Errorf("expected state=open, got %s", r.URL.Query().Get("state"))
				}
				if r.URL.Query().Get("per_page") != "30" {
					t.Errorf("expected per_page=30, got %s", r.URL.Query().Get("per_page"))
				}

				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode([]map[string]interface{}{
					{"id": 12345, "number": 1, "title": "First issue", "html_url": "https://github.com/testowner/testrepo/issues/1"},
					{"id": 12346, "number": 2, "title": "Second issue", "html_url": "https://github.com/testowner/testrepo/issues/2"},
				})
			},
			wantCount: 2,
			wantErr:   false,
		},
		{
			name: "empty list",
			opts: ListIssuesOptions{
				Owner:   "testowner",
				Repo:    "testrepo",
				State:   "open",
				PerPage: 30,
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode([]map[string]interface{}{})
			},
			wantCount: 0,
			wantErr:   false,
		},
		{
			name: "API error",
			opts: ListIssuesOptions{
				Owner:   "testowner",
				Repo:    "testrepo",
				State:   "open",
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.serverResponse))
			defer server.Close()

			client := &Client{
				HTTPClient: server.Client(),
				BaseURL:    server.URL,
			}

			issues, err := client.ListIssues(tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("ListIssues() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && len(issues) != tt.wantCount {
				t.Errorf("ListIssues() returned %d issues, want %d", len(issues), tt.wantCount)
			}
		})
	}
}

func TestGetIssue(t *testing.T) {
	tests := []struct {
		name           string
		owner          string
		repo           string
		number         int
		serverResponse func(w http.ResponseWriter, r *http.Request)
		wantErr        bool
	}{
		{
			name:   "gets existing issue",
			owner:  "testowner",
			repo:   "testrepo",
			number: 42,
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" {
					t.Errorf("expected GET, got %s", r.Method)
				}
				if r.URL.Path != "/repos/testowner/testrepo/issues/42" {
					t.Errorf("unexpected path: %s", r.URL.Path)
				}

				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"id":     99999,
					"number": 42,
					"title":  "Parent Issue",
				})
			},
			wantErr: false,
		},
		{
			name:   "handles not found",
			owner:  "testowner",
			repo:   "testrepo",
			number: 999,
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"message": "Not Found",
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

			_, err := client.GetIssue(tt.owner, tt.repo, tt.number)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetIssue() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
