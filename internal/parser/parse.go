package parser

// Parser is the interface every parser must satisfy
type Parser interface {
	Parse(path string) (Mapped, error)
}

type StepItem struct {
	ID    string            `json:"id"`
	Label string            `json:"label"`
	Kind  string            `json:"kind"`
	Attrs map[string]string `json:"attrs"`
}

type PrincipalItem struct {
	ID    string            `json:"id"`
	Label string            `json:"label"`
	Kind  string            `json:"kind"`
	Attrs map[string]string `json:"attrs"`
}

type ArtifactItem struct {
	ID    string            `json:"id"`
	Label string            `json:"label"`
	Kind  string            `json:"kind"`
	Attrs map[string]string `json:"attrs"`
}

type ResourceItem struct {
	ID    string            `json:"id"`
	Label string            `json:"label"`
	Kind  string            `json:"kind"`
	Attrs map[string]string `json:"attrs"`
}

// Record holds one unit of parsed provenance
type Record struct {
	Step         StepItem       `json:"step"`
	Principal    PrincipalItem  `json:"principal"`
	ArtifactsIn  []ArtifactItem `json:"artifacts_in"`
	ArtifactsOut []ArtifactItem `json:"artifacts_out"`
	Resources    []ResourceItem `json:"resources"`
}

// Mapped is the top-level output of a parser: Records plus metadata.
type Mapped struct {
	Mapped       []Record `json:"mapped"`
	Source       string   `json:"source"`
	NormalizedAt int64    `json:"normalized_at"`
}
