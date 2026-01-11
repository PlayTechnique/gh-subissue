package cmd

import (
	"fmt"

	"github.com/gwyn/gh-subissue/internal/api"
	"github.com/gwyn/gh-subissue/internal/debug"
)

// SelectParentIssue prompts user to select an issue from a list.
// Returns the selected issue number.
func SelectParentIssue(p Prompter, issues []api.Issue) (int, error) {
	debug.Log("SelectParentIssue", "issue_count", len(issues))

	if len(issues) == 0 {
		err := fmt.Errorf("no open issues found in repository\n\nTo create a parent issue first:\n  gh issue create\n\nOr use --parent with an existing issue number")
		debug.Error("SelectParentIssue", err)
		return 0, err
	}

	options := make([]string, len(issues))
	for i, issue := range issues {
		// Format: "#123 Issue title" (truncate title if needed)
		title := issue.Title
		if len(title) > 50 {
			title = title[:47] + "..."
		}
		options[i] = fmt.Sprintf("#%d %s", issue.Number, title)
	}

	debug.Log("SelectParentIssue", "action", "prompting_user", "options_count", len(options))
	idx, err := p.Select("Select parent issue", "", options)
	if err != nil {
		debug.Error("SelectParentIssue", err, "stage", "prompt_select")
		return 0, err
	}

	selectedNumber := issues[idx].Number
	debug.Log("SelectParentIssue", "selected_index", idx, "selected_number", selectedNumber)
	return selectedNumber, nil
}
