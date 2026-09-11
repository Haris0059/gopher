// Package handlers implements CLI subcommand handlers.
package handlers

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/Haris0059/gopher/pkg/skills"
)

// AgentsHandler prints the list of configured agents grouped by source.
// Output goes to w. cwd is the project working directory used to locate
// project-local agent definitions.
func AgentsHandler(w io.Writer, cwd string) {
	all := skills.LoadAgents(cwd)
	printAgents(w, all)
}

// AgentsHandlerWithDirs is like AgentsHandler but accepts explicit directory
// mappings (plus the built-in agents). Useful for testing without touching
// the real filesystem.
func AgentsHandlerWithDirs(w io.Writer, dirs map[skills.AgentSource]string) {
	all := skills.GetBuiltInAgents()
	all = append(all, skills.LoadAgentsFromDirs(dirs)...)
	printAgents(w, all)
}

func printAgents(w io.Writer, all []skills.AgentDefinition) {
	active := skills.GetActiveAgents(all)
	resolved := skills.ResolveAgentOverrides(all, active)

	var lines []string
	totalActive := 0

	for _, sg := range skills.AgentSourceGroups {
		// Filter to this source group.
		var group []skills.ResolvedAgent
		for _, r := range resolved {
			if r.Source == sg.Source {
				group = append(group, r)
			}
		}
		if len(group) == 0 {
			continue
		}
		sort.Slice(group, func(i, j int) bool {
			return skills.CompareAgentsByName(group[i].AgentDefinition, group[j].AgentDefinition) < 0
		})

		lines = append(lines, sg.Label+":")
		for _, a := range group {
			if a.OverriddenBy != "" {
				label := skills.OverrideSourceLabel(a.OverriddenBy)
				lines = append(lines, fmt.Sprintf("  (shadowed by %s) %s", label, skills.FormatAgentListing(a)))
			} else {
				lines = append(lines, "  "+skills.FormatAgentListing(a))
				totalActive++
			}
		}
		lines = append(lines, "")
	}

	if len(lines) == 0 {
		fmt.Fprintln(w, "No agents found.")
		return
	}

	fmt.Fprintf(w, "%d active agents\n\n", totalActive)
	fmt.Fprintln(w, strings.TrimRight(strings.Join(lines, "\n"), "\n"))
}
