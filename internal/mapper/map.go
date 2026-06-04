package mapper

import (
	"github.com/TSELab/astra/internal/graph"
	"github.com/TSELab/astra/internal/parser"
)

// ToAstraGraph converts parser.Mapped ([]Record) into a typed graph.AstraGraph.
// Relations emitted:
//
//	principal --uses--> resource
//	resource  --carries_out--> step
//	step  --consumes--> artifact
//	step      --produces--> artifact
func ToAstraGraph(m parser.Mapped) graph.AstraGraph {
	out := graph.NewAstraGraph()

	edgeSet := map[string]bool{}
	addEdge := func(src, dst, rel string, md map[string]string) {
		if src == "" || dst == "" || rel == "" {
			return
		}
		k := src + "|" + rel + "|" + dst
		if edgeSet[k] {
			return
		}
		edgeSet[k] = true
		out.Edges = append(out.Edges, graph.Edge{
			Source:   src,
			Target:   dst,
			Relation: rel,
		})
		_ = md
	}

	for _, rec := range m.Mapped {
		if rec.Principal.ID != "" {
			p := rec.Principal
			completeness := p.Completeness
			if completeness == "" {
				completeness = graph.Complete
			}
			if existing, ok := out.Principals[p.ID]; !ok || graph.CompletenessRank(completeness) > graph.CompletenessRank(existing.Completeness) {
				md := cloneMap(p.Attrs)
				if md == nil {
					md = map[string]string{}
				}
				out.Principals[p.ID] = graph.Principal{
					ID:           p.ID,
					Trust:        "unknown",
					Builder:      "",
					Name:         p.Label,
					Metadata:     md,
					Completeness: completeness,
				}
			}
		}

		if rec.Step.ID != "" {
			s := rec.Step
			completeness := s.Completeness
			if completeness == "" {
				completeness = graph.Complete
			}
			if existing, ok := out.Steps[s.ID]; !ok || graph.CompletenessRank(completeness) > graph.CompletenessRank(existing.Completeness) {
				md := cloneMap(s.Attrs)
				if md == nil {
					md = map[string]string{}
				}
				out.Steps[s.ID] = graph.Step{
					ID:           s.ID,
					Command:      normalizeStepCommand(md),
					Timestamp:    normalizeTimestamp(md),
					Arch:         normalizeStepArch(md),
					Environment:  map[string]string{},
					Metadata:     md,
					Completeness: completeness,
				}
			}
		}

		for _, r := range rec.Resources {
			if r.ID == "" {
				continue
			}
			completeness := r.Completeness
			if completeness == "" {
				completeness = graph.Complete
			}
			if existing, ok := out.Resources[r.ID]; !ok || graph.CompletenessRank(completeness) > graph.CompletenessRank(existing.Completeness) {
				out.Resources[r.ID] = graph.Resource{
					ID:           r.ID,
					Type:         normalizeResourceType(r),
					URI:          normalizeResourceURI(r),
					Format:       normalizeResourceFormat(r),
					Metadata:     cloneMap(r.Attrs),
					Completeness: completeness,
				}
			}
		}

		for _, it := range rec.ArtifactsIn {
			if it.ID == "" {
				continue
			}

			if _, ok := out.Artifacts[it.ID]; !ok {
				out.Artifacts[it.ID] = normalizeArtifact(it)
			}

			addEdge(it.ID, rec.Step.ID, "consumes", nil)
		}

		for _, it := range rec.ArtifactsOut {
			if it.ID == "" {
				continue
			}

			if _, ok := out.Artifacts[it.ID]; !ok {
				out.Artifacts[it.ID] = normalizeArtifact(it)
			}

			addEdge(rec.Step.ID, it.ID, "produces", nil)
		}

		for _, r := range rec.Resources {
			addEdge(rec.Principal.ID, r.ID, "uses", nil)
			addEdge(r.ID, rec.Step.ID, "carries_out", nil)
		}
		if rec.Principal.ID != "" && rec.Step.ID != "" {
			addEdge(rec.Principal.ID, rec.Step.ID, "performs", nil)
		}

		for _, dep := range rec.Dependencies {
			if dep.ID == "" {
				continue
			}
			if _, ok := out.Artifacts[dep.ID]; !ok {
				out.Artifacts[dep.ID] = normalizeArtifact(dep)
			}
			// depends edges go from each output artifact to the dependency.
			// Direction: openssh-server --depends--> libsystemd0 --depends--> liblzma5
			for _, out := range rec.ArtifactsOut {
				addEdge(out.ID, dep.ID, "depends", nil)
			}
		}
	}

	return out
}
