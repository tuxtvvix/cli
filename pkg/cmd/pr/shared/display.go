package shared

import (
	"fmt"

	"github.com/cli/cli/v2/api"
	"github.com/cli/cli/v2/internal/text"
	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/cli/cli/v2/pkg/iostreams"
)

func StateTitleWithColor(cs *iostreams.ColorScheme, pr api.PullRequest) string {
	prStateColorFunc := cs.ColorFromString(ColorForPRState(pr))
	if pr.State == "OPEN" && pr.IsDraft {
		return prStateColorFunc("Draft")
	}
	return prStateColorFunc(text.Title(pr.State))
}

// PRNumberWithColor returns a colored string representation of a pull request number
// based on its state (open, closed, merged, or draft).
// It prefixes the number with a hash symbol (#) to indicate it's a pull request.
func PRNumberWithColor(cs *iostreams.ColorScheme, pr api.PullRequest) string {
	prStateColorFunc := cs.ColorFromRGB(ColorHexCodeForPRState(pr))
	prNumber := fmt.Sprintf("#%d", pr.Number)
	return prStateColorFunc(text.Title(prNumber))
}

func ColorForPRState(pr api.PullRequest) string {
	switch pr.State {
	case "OPEN":
		if pr.IsDraft {
			return "gray"
		}
		return "green"
	case "CLOSED":
		return "red"
	case "MERGED":
		return "magenta"
	default:
		return ""
	}
}

func ColorHexCodeForPRState(pr api.PullRequest) string {
	switch pr.State {
	case "OPEN":
		if pr.IsDraft {
			return "808080"
		}
		return "00FF00"
	case "CLOSED":
		return "FF0000"
	case "MERGED":
		return "FFA500"
	default:
		return ""
	}
}

func ColorForIssueState(issue api.Issue) string {
	switch issue.State {
	case "OPEN":
		return "green"
	case "CLOSED":
		if issue.StateReason == "NOT_PLANNED" {
			return "gray"
		}
		return "magenta"
	default:
		return ""
	}
}

func PrintHeader(io *iostreams.IOStreams, s string) {
	fmt.Fprintln(io.Out, io.ColorScheme().Bold(s))
}

func PrintMessage(io *iostreams.IOStreams, s string) {
	fmt.Fprintln(io.Out, io.ColorScheme().Gray(s))
}

func ListNoResults(repoName string, itemName string, hasFilters bool) error {
	if hasFilters {
		return cmdutil.NewNoResultsError(fmt.Sprintf("no %ss match your search in %s", itemName, repoName))
	}
	return cmdutil.NewNoResultsError(fmt.Sprintf("no open %ss in %s", itemName, repoName))
}

func ListHeader(repoName string, itemName string, matchCount int, totalMatchCount int, hasFilters bool) string {
	if hasFilters {
		matchVerb := "match"
		if totalMatchCount == 1 {
			matchVerb = "matches"
		}
		return fmt.Sprintf("Showing %d of %s in %s that %s your search", matchCount, text.Pluralize(totalMatchCount, itemName), repoName, matchVerb)
	}

	return fmt.Sprintf("Showing %d of %s in %s", matchCount, text.Pluralize(totalMatchCount, fmt.Sprintf("open %s", itemName)), repoName)
}

func PrCheckStatusSummaryWithColor(cs *iostreams.ColorScheme, checks api.PullRequestChecksStatus) string {
	var summary = cs.Gray("No checks")
	if checks.Total > 0 {
		if checks.Failing > 0 {
			if checks.Failing == checks.Total {
				summary = cs.Red("× All checks failing")
			} else {
				summary = cs.Redf("× %d/%d checks failing", checks.Failing, checks.Total)
			}
		} else if checks.Pending > 0 {
			summary = cs.Yellow("- Checks pending")
		} else if checks.Passing == checks.Total {
			summary = cs.Green("✓ Checks passing")
		}
	}
	return summary
}
