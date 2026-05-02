package lore

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// BranchGroup represents a group of sessions under one branch
type BranchGroup struct {
	Branch   string
	Sessions []Session
}

// groupByBranch groups sessions by branch, with each group's sessions sorted by
// timestamp descending (newest first). Returns groups sorted by the timestamp of
// the most recent session in each group (descending).
func groupByBranch(sessions []Session) []BranchGroup {
	if len(sessions) == 0 {
		return nil
	}

	// Group sessions by branch
	groupMap := make(map[string][]Session)
	for _, s := range sessions {
		groupMap[s.Branch] = append(groupMap[s.Branch], s)
	}

	// Create BranchGroup slice and sort sessions within each group
	var groups []BranchGroup
	for branch, sessionsInGroup := range groupMap {
		// Sort sessions in this group by timestamp (newest first)
		sort.Slice(sessionsInGroup, func(i, j int) bool {
			return sessionsInGroup[i].Timestamp.After(sessionsInGroup[j].Timestamp)
		})
		groups = append(groups, BranchGroup{Branch: branch, Sessions: sessionsInGroup})
	}

	// Sort groups by the timestamp of their most recent session (newest first)
	sort.Slice(groups, func(i, j int) bool {
		if len(groups[i].Sessions) == 0 || len(groups[j].Sessions) == 0 {
			return false
		}
		return groups[i].Sessions[0].Timestamp.After(groups[j].Sessions[0].Timestamp)
	})

	return groups
}

// renderProjectView renders the project view
func renderProjectView(m model, now time.Time) string {
	if len(m.projectSessions) == 0 {
		var b strings.Builder
		b.WriteString(renderProjectHeader(m))
		b.WriteByte('\n')
		b.WriteString(renderDivider(m.width))
		b.WriteByte('\n')
		b.WriteString(" (no sessions for this project)\n")
		b.WriteString(renderDivider(m.width))
		b.WriteByte('\n')
		b.WriteString(renderProjectFooter())
		b.WriteByte('\n')
		return b.String()
	}

	var b strings.Builder
	b.WriteString(renderProjectHeader(m))
	b.WriteByte('\n')
	b.WriteString(renderDivider(m.width))
	b.WriteByte('\n')

	// Group by branch and render
	groups := groupByBranch(m.projectSessions)

	sessionIdx := 0
	for _, group := range groups {
		// Render branch heading
		latestInGroup := group.Sessions[0]
		bucketLabel := timeBucket(latestInGroup.Timestamp, now)
		headingLine := fmt.Sprintf(" %s   %d session%s   %s",
			group.Branch,
			len(group.Sessions),
			plural(len(group.Sessions)),
			bucketLabel,
		)
		b.WriteString(bucketStyle.Render(headingLine))
		b.WriteByte('\n')

		// Render sessions in this group
		for _, sess := range group.Sessions {
			isSelected := (sessionIdx == m.projectCursor)
			line := fmt.Sprintf("  %s  %s",
				sess.Timestamp.Format("15:04"),
				sess.Slug,
			)
			if isSelected {
				b.WriteString(selectedStyle.Render(line))
			} else {
				b.WriteString(line)
			}
			b.WriteByte('\n')
			sessionIdx++
		}
	}

	b.WriteString(renderDivider(m.width))
	b.WriteByte('\n')
	b.WriteString(renderProjectFooter())
	b.WriteByte('\n')

	return b.String()
}

func renderProjectHeader(m model) string {
	return headerStyle.Render(fmt.Sprintf(" %s · %s   %d session%s",
		m.sessions[m.cursor].Project,
		m.projectCWD,
		len(m.projectSessions),
		plural(len(m.projectSessions)),
	))
}

func renderProjectFooter() string {
	return footerStyle.Render(" j/k move   enter open   q/esc back")
}
