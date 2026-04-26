package printer

import (
	"fmt"
	"strings"
	"time"
)

// FormatRoll returns a single receipt string for a roll event.
// Goes in full to the stext field.
func FormatRoll(username, producer, productBold, productThin, country, status string) string {
	var sb strings.Builder
	//sb.WriteString("\n\n")
	sb.WriteString(time.Now().Format("2006-01-02 15:04:05"))
	sb.WriteString("\n")
	sb.WriteString(username)
	sb.WriteString(" ")
	sb.WriteString(statusVerb(status))
	sb.WriteString("\n")
	sb.WriteString(strings.TrimSpace(productBold + " " + productThin))
	sb.WriteString("\n")
	sb.WriteString(producer)
	sb.WriteString("\n")
	sb.WriteString(country)
	return sb.String()
}

// FormatStatus returns a short slip noting a user's accept/veto outcome,
// with a duration line showing how long the decision took.
// Leading/trailing blank lines give visual breathing room above and below.
// decisionSeconds is the persisted accept/veto duration (1-decimal seconds);
// nil omits the duration line.
func FormatStatus(username, action string, decisionSeconds *float64) string {
	var sb strings.Builder
	sb.WriteString("\n\n")
	sb.WriteString(username)
	sb.WriteString(" ")
	sb.WriteString(action)
	if dur := formatDecisionSeconds(decisionSeconds); dur != "" {
		sb.WriteString("\nTime until ")
		sb.WriteString(action)
		sb.WriteString(": ")
		sb.WriteString(dur)
	}
	sb.WriteString("\n\n")
	return sb.String()
}

func formatDecisionSeconds(s *float64) string {
	if s == nil || *s < 0 {
		return ""
	}
	total := *s
	h := int(total) / 3600
	total -= float64(h * 3600)
	m := int(total) / 60
	secs := total - float64(m*60)

	switch {
	case h > 0:
		return fmt.Sprintf("%dh %dm %.1fs", h, m, secs)
	case m > 0:
		return fmt.Sprintf("%dm %.1fs", m, secs)
	default:
		return fmt.Sprintf("%.1fs", secs)
	}
}

func statusVerb(status string) string {
	switch status {
	case "pending":
		return "rolled"
	case "accepted":
		return "accepted"
	case "vetoed":
		return "vetoed"
	default:
		return status
	}
}
