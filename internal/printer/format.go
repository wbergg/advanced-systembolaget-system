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
func FormatStatus(username, action, createdAt, resolvedAt string) string {
	var sb strings.Builder
	sb.WriteString("\n\n")
	sb.WriteString(username)
	sb.WriteString(" ")
	sb.WriteString(action)
	if dur := decisionDuration(createdAt, resolvedAt); dur != "" {
		sb.WriteString("\nTime until ")
		sb.WriteString(action)
		sb.WriteString(": ")
		sb.WriteString(dur)
	}
	sb.WriteString("\n\n")
	return sb.String()
}

func decisionDuration(from, to string) string {
	if from == "" || to == "" {
		return ""
	}
	t1, err1 := time.Parse(time.RFC3339, from)
	t2, err2 := time.Parse(time.RFC3339, to)
	if err1 != nil || err2 != nil {
		return ""
	}
	return formatDuration(t2.Sub(t1))
}

func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second

	switch {
	case h > 0:
		return fmt.Sprintf("%dh %dm %ds", h, m, s)
	case m > 0:
		return fmt.Sprintf("%dm %ds", m, s)
	default:
		return fmt.Sprintf("%ds", s)
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
