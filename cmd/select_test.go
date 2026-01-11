package cmd

import (
	"errors"
	"testing"

	"github.com/gwyn/gh-subissue/internal/api"
)

// mockPrompter implements Prompter for testing.
type mockPrompter struct {
	selectFunc func(prompt string, defaultValue string, options []string) (int, error)
}

func (m *mockPrompter) Select(prompt string, defaultValue string, options []string) (int, error) {
	if m.selectFunc != nil {
		return m.selectFunc(prompt, defaultValue, options)
	}
	return 0, nil
}

// Compile-time check: mockPrompter must implement Prompter
var _ Prompter = (*mockPrompter)(nil)

func TestSelectParentIssue(t *testing.T) {
	tests := []struct {
		name       string
		issues     []api.Issue
		selectIdx  int
		selectErr  error
		wantNumber int
		wantErr    bool
	}{
		{
			name: "selects first issue",
			issues: []api.Issue{
				{ID: 100, Number: 10, Title: "First issue"},
				{ID: 200, Number: 20, Title: "Second issue"},
			},
			selectIdx:  0,
			wantNumber: 10,
			wantErr:    false,
		},
		{
			name: "selects second issue",
			issues: []api.Issue{
				{ID: 100, Number: 10, Title: "First issue"},
				{ID: 200, Number: 20, Title: "Second issue"},
			},
			selectIdx:  1,
			wantNumber: 20,
			wantErr:    false,
		},
		{
			name:       "empty issue list returns error",
			issues:     []api.Issue{},
			wantErr:    true,
		},
		{
			name: "prompter error propagates",
			issues: []api.Issue{
				{ID: 100, Number: 10, Title: "First issue"},
			},
			selectErr: errors.New("user cancelled"),
			wantErr:   true,
		},
		{
			name: "long title is truncated",
			issues: []api.Issue{
				{ID: 100, Number: 10, Title: "This is a very long title that should be truncated because it exceeds fifty characters"},
			},
			selectIdx:  0,
			wantNumber: 10,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedOptions []string
			p := &mockPrompter{
				selectFunc: func(prompt string, defaultValue string, options []string) (int, error) {
					capturedOptions = options
					if tt.selectErr != nil {
						return 0, tt.selectErr
					}
					return tt.selectIdx, nil
				},
			}

			num, err := SelectParentIssue(p, tt.issues)
			if (err != nil) != tt.wantErr {
				t.Errorf("SelectParentIssue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if num != tt.wantNumber {
					t.Errorf("SelectParentIssue() = %v, want %v", num, tt.wantNumber)
				}
			}

			// Check title truncation for the long title test
			if tt.name == "long title is truncated" && len(capturedOptions) > 0 {
				opt := capturedOptions[0]
				// Format is "#N title..." so check the truncated part
				if len(opt) > 60 {
					t.Errorf("Option too long: %d chars, got %q", len(opt), opt)
				}
				if opt[len(opt)-3:] != "..." {
					t.Errorf("Expected truncated title to end with '...', got %q", opt)
				}
			}
		})
	}
}
