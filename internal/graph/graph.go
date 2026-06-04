// internal/graph/graph.go
package graph

import (
	"fmt"
	"sort"
	"strings"
)

// ToDOT renders an AstraGraph into Graphviz DOT format.
//
// Output file artifacts (git-file nodes that are only ever produced, never
// consumed) are omitted: if file A was consumed, it is self-evident that the
// step produced a new version of file A. Only consumed file artifacts are
// shown, keeping the graph linear: commit → step → commit → …
func ToDOT(g AstraGraph) string {
	// Build the set of git-file artifact IDs that appear as inputs (consumed).
	// These are the only file artifacts we render.
	consumedFiles := map[string]bool{}
	for _, e := range g.Edges {
		if e.Relation == "consumes" {
			if a, ok := g.Artifacts[e.Source]; ok && a.Kind == "git-file" {
				consumedFiles[e.Source] = true
			}
		}
	}

	// Filter edges: drop "produces" edges whose target is a git-file artifact.
	var edges []Edge
	for _, e := range g.Edges {
		if e.Relation == "produces" {
			if a, ok := g.Artifacts[e.Target]; ok && a.Kind == "git-file" {
				continue
			}
		}
		edges = append(edges, e)
	}

	// Stable sort for deterministic output.
	sort.Slice(edges, func(i, j int) bool {
		ei, ej := edges[i], edges[j]
		if ei.Source != ej.Source {
			return ei.Source < ej.Source
		}
		if ei.Relation != ej.Relation {
			return ei.Relation < ej.Relation
		}
		return ei.Target < ej.Target
	})

	var b strings.Builder
	b.WriteString("digraph astra {\n")

	// Artifacts: commit artifacts always shown; file artifacts only if consumed.
	artifactIDs := make([]string, 0, len(g.Artifacts))
	for id := range g.Artifacts {
		artifactIDs = append(artifactIDs, id)
	}
	sort.Strings(artifactIDs)
	for _, id := range artifactIDs {
		n := g.Artifacts[id]
		if n.Kind == "git-file" && !consumedFiles[id] {
			continue
		}
		var label string
		if n.Kind == "git-commit" && n.Version != "" {
			short := n.Version
			if len(short) > 8 {
				short = short[:8]
			}
			label = "commit:" + short
		} else {
			label = n.Name
			if n.Version != "" {
				short := n.Version
				if len(short) > 8 {
					short = short[:8]
				}
				label = label + "@" + short
			}
		}
		fmt.Fprintf(&b, "  \"%s\" [label=\"%s\\n(Artifact)\" shape=box];\n", id, label)
	}

	// Steps
	stepIDs := make([]string, 0, len(g.Steps))
	for id := range g.Steps {
		stepIDs = append(stepIDs, id)
	}
	sort.Strings(stepIDs)
	for _, id := range stepIDs {
		n := g.Steps[id]
		label := n.Command
		if label == "" {
			if msg, ok := n.Metadata["message"]; ok && msg != "" {
				if i := strings.Index(msg, "\n"); i >= 0 {
					msg = msg[:i]
				}
				if len(msg) > 40 {
					msg = msg[:40] + "..."
				}
				label = msg
			}
		}
		fmt.Fprintf(&b, "  \"%s\" [label=\"%s\\n(Step)\" shape=diamond];\n", id, label)
	}

	// Principals
	principalIDs := make([]string, 0, len(g.Principals))
	for id := range g.Principals {
		principalIDs = append(principalIDs, id)
	}
	sort.Strings(principalIDs)
	for _, id := range principalIDs {
		n := g.Principals[id]
		fmt.Fprintf(&b, "  \"%s\" [label=\"%s\\n(Principal)\" shape=oval];\n", id, n.Name)
	}

	// Resources
	resourceIDs := make([]string, 0, len(g.Resources))
	for id := range g.Resources {
		resourceIDs = append(resourceIDs, id)
	}
	sort.Strings(resourceIDs)
	for _, id := range resourceIDs {
		fmt.Fprintf(&b, "  \"%s\" [label=\"%s\\n(Resource)\" shape=hexagon];\n", id, id)
	}

	// Edges
	for _, e := range edges {
		fmt.Fprintf(&b, "  \"%s\" -> \"%s\" [label=\"%s\"];\n", e.Source, e.Target, e.Relation)
	}
	b.WriteString("}\n")
	return b.String()
}

// constructor to generate new AstraGraph
func NewAstraGraph() AstraGraph {
	return AstraGraph{
		Artifacts:  map[string]Artifact{},
		Steps:      map[string]Step{},
		Principals: map[string]Principal{},
		Resources:  map[string]Resource{},
		Edges:      []Edge{},
	}
}

// ExportGraph is used for JSON output / deterministic ordering
type ExportGraph struct {
	Artifacts  []Artifact  `json:"artifacts"`
	Steps      []Step      `json:"steps"`
	Principals []Principal `json:"principals"`
	Resources  []Resource  `json:"resources"`
	Edges      []Edge      `json:"edges"`
}

// ToExport converts AstraGraph to ExportGraph that is used for JSON output / deterministic ordering
func ToExport(g AstraGraph) ExportGraph {
	var out ExportGraph

	// Artifacts
	for _, a := range g.Artifacts {
		out.Artifacts = append(out.Artifacts, a)
	}
	sort.Slice(out.Artifacts, func(i, j int) bool {
		return out.Artifacts[i].ID < out.Artifacts[j].ID
	})

	// Steps
	for _, s := range g.Steps {
		out.Steps = append(out.Steps, s)
	}
	sort.Slice(out.Steps, func(i, j int) bool {
		return out.Steps[i].ID < out.Steps[j].ID
	})

	// Principals
	for _, p := range g.Principals {
		out.Principals = append(out.Principals, p)
	}
	sort.Slice(out.Principals, func(i, j int) bool {
		return out.Principals[i].ID < out.Principals[j].ID
	})

	// Resources
	for _, r := range g.Resources {
		out.Resources = append(out.Resources, r)
	}
	sort.Slice(out.Resources, func(i, j int) bool {
		return out.Resources[i].ID < out.Resources[j].ID
	})

	// Edges
	for _, e := range g.Edges {
		out.Edges = append(out.Edges, e)
	}
	sort.Slice(out.Edges, func(i, j int) bool {
		if out.Edges[i].Source != out.Edges[j].Source {
			return out.Edges[i].Source < out.Edges[j].Source
		}
		if out.Edges[i].Relation != out.Edges[j].Relation {
			return out.Edges[i].Relation < out.Edges[j].Relation
		}
		return out.Edges[i].Target < out.Edges[j].Target
	})

	return out
}

// FromExport convert ExportGraph to AstraGraph, used to visualize/analyze json format graphs
func FromExport(e ExportGraph) AstraGraph {
	g := NewAstraGraph()

	for _, a := range e.Artifacts {
		g.Artifacts[a.ID] = a
	}
	for _, s := range e.Steps {
		g.Steps[s.ID] = s
	}
	for _, p := range e.Principals {
		g.Principals[p.ID] = p
	}
	for _, r := range e.Resources {
		g.Resources[r.ID] = r
	}
	for _, ed := range e.Edges {
		g.Edges = append(g.Edges, ed)
	}

	return g
}

// CompletenessRank returns a numeric rank for ordering completeness states.
// complete(1) outranks incomplete(0).
func CompletenessRank(c string) int {
	if c == Complete {
		return 1
	}
	return 0
}

// Merge combines two AstraGraph structs into one.
// Nodes are merged by ID with completeness-aware upgrade:
// an incoming node replaces a stored node only when it is more complete.
// Edges are deduplicated by source|relation|target.
func Merge(a, b AstraGraph) AstraGraph {
	for id, v := range b.Artifacts {
		if existing, ok := a.Artifacts[id]; !ok || CompletenessRank(v.Completeness) > CompletenessRank(existing.Completeness) {
			a.Artifacts[id] = v
		}
	}
	for id, v := range b.Steps {
		if existing, ok := a.Steps[id]; !ok || CompletenessRank(v.Completeness) > CompletenessRank(existing.Completeness) {
			a.Steps[id] = v
		}
	}
	for id, v := range b.Resources {
		if existing, ok := a.Resources[id]; !ok || CompletenessRank(v.Completeness) > CompletenessRank(existing.Completeness) {
			a.Resources[id] = v
		}
	}
	for id, v := range b.Principals {
		if existing, ok := a.Principals[id]; !ok || CompletenessRank(v.Completeness) > CompletenessRank(existing.Completeness) {
			a.Principals[id] = v
		}
	}

	seen := map[string]bool{}
	for _, e := range a.Edges {
		seen[e.Source+"|"+e.Relation+"|"+e.Target] = true
	}
	for _, e := range b.Edges {
		k := e.Source + "|" + e.Relation + "|" + e.Target
		if !seen[k] {
			seen[k] = true
			a.Edges = append(a.Edges, e)
		}
	}

	return a
}

//TODO
/*
Validates DAG properties
- Detect and reject cycles in the AStRA graph
- Enforce required structural invariants, e.g.:
		every Artifact has ≥1 producing Step
		every Step is associated with ≥1 Principal and Resource
		Ensure edge directions respect causal semantics

Add temporal reasoning
- Validate that edge directions are consistent with timestamps
- Detect temporal anomalies (e.g., artifact consumed before production)
- Enable reasoning about:
- Check step ordering -> run step to step order and verify the temporal order

*/
