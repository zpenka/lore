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

// projectBodyLines builds the rendered branch-grouped session rows for
// project mode. cursorLine is the line index of the row that holds the
// projectCursor's selected session (which indexes into the flat list).
func projectBodyLines(m model, now time.Time) (lines []string, cursorLine int) {
	groups := groupByBranch(m.projectSessions)
	sessionIdx := 0
	for _, group := range groups {
		latestInGroup := group.Sessions[0]
		bucketLabel := timeBucket(latestInGroup.Timestamp, now)
		headingLine := fmt.Sprintf(" %s   %d session%s   %s",
			group.Branch,
			len(group.Sessions),
			plural(len(group.Sessions)),
			bucketLabel,
		)
		lines = append(lines, bucketStyle.Render(headingLine))

		for _, sess := range group.Sessions {
			isSelected := (sessionIdx == m.projectCursor)
			label := sess.Query
			if label == "" {
				label = sess.Slug
			}
			row := fmt.Sprintf("  %s  %s",
				sess.Timestamp.Format("15:04"),
				label,
			)
			if isSelected {
				cursorLine = len(lines)
				lines = append(lines, selectedStyle.Render(row))
			} else {
				lines = append(lines, row)
			}
			sessionIdx++
		}
	}
	return
}

// renderProjectView renders the project view
func renderProjectView(m model, now time.Time) string {
	var b strings.Builder
	b.WriteString(renderProjectHeader(m))
	b.WriteByte('\n')
	b.WriteString(renderDivider(m.width))
	b.WriteByte('\n')

	body, cursorLine := projectBodyLines(m, now)
	if len(body) == 0 {
		body = []string{" (no sessions for this project)"}
		cursorLine = 0
	}
	height := m.bodyHeight()
	offset := clampOffset(m.projectOffset, cursorLine, len(body), height)
	for _, line := range renderBody(body, offset, height) {
		b.WriteString(line)
		b.WriteByte('\n')
	}

	b.WriteString(renderDivider(m.width))
	b.WriteByte('\n')
	b.WriteString(renderProjectFooter(m))
	b.WriteByte('\n')

	return b.String()
}

func renderProjectHeader(m model) string {
	// Derive the displayed project name from the projectCWD itself rather
	// than indexing back into m.sessions[m.cursor], which can be wrong if
	// the user navigated through search or some other entry path.
	name := m.projectCWD
	if i := strings.LastIndex(name, "/"); i >= 0 && i < len(name)-1 {
		name = name[i+1:]
	}
	return headerStyle.Render(fmt.Sprintf(" %s · %s   %d session%s",
		name,
		m.projectCWD,
		len(m.projectSessions),
		plural(len(m.projectSessions)),
	))
}

func renderProjectFooter(m model) string {
	if m.flashMsg != "" {
		return flashStyle.Render(" " + m.flashMsg)
	}
	return footerStyle.Render(" j/k move   d/u page   enter open   g/G top/bottom   q/esc/h/← back")
}
