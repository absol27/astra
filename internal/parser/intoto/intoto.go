/*
Package intoto implements the InTotoParser for AStRA.

InTotoParser parses in-toto link files and maps each signed step into
AStRA's record structure. Each link file produces one Record.

For an in-toto link file:
  - The signed step is represented as a step node with kind "linker",
    ID scheme step:linker:<linker-name>:<step-name>@<hash-of-signed-block>.
    The hash makes the step content-addressed: the same link file always
    produces the same step ID.
  - The first signing key identifies the principal; unsigned links use
    principal:intoto:unsigned.
  - Materials (inputs) become input artifacts with completeness "incomplete"
    — only the hash is known, not the full content.
  - Products (outputs) become output artifacts with completeness "complete"
    — the link cryptographically attests to their content.

Artifact IDs are resolved from filenames in priority order:
  1. <pkg>_<ver>_<arch>.deb         → artifact:pkg:deb/debian/<pkg>@<ver>?arch=<arch>
  2. <src>_<upstream>.orig.tar.<ext> → artifact:tarball:deb/<src>@<upstream>
  3. <src>_<ver>_<arch>.buildinfo   → artifact:buildinfo:deb/<src>@<ver>?arch=<arch>
  4. <host>/<owner>/<repo>@<ref>    → artifact:gitcommit:<slug>@<sha256-field>
                                       (sha256 field carries the git commit hash)
  5. anything else                  → artifact:intoto:<step>:<role>:<sha256>

IDs 1–3 are identical to those produced by the buildinfo and packages parsers,
so shared artifacts merge automatically when multiple documents are ingested.
*/
package intoto

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	parser "github.com/TSELab/astra/internal/parser"
	"github.com/TSELab/astra/internal/parser/debian/debutil"
)

// InTotoParser implements parser.Parser for in-toto link files.
// Linker names the provenance database that generated the link files and
// is embedded in the step ID as step:linker:<Linker>:<name>@<hash>.
// Defaults to "debtrace" if empty.
type InTotoParser struct {
	Linker string
}

// linkFile is the top-level JSON structure of an in-toto link file.
type linkFile struct {
	Signed     signedLink  `json:"signed"`
	Signatures []signature `json:"signatures"`
}

// signedLink is the "signed" block within an in-toto link file.
type signedLink struct {
	Type        string                 `json:"_type"`
	Name        string                 `json:"name"`
	Materials   map[string]hashDict    `json:"materials"`
	Products    map[string]hashDict    `json:"products"`
	ByProducts  map[string]interface{} `json:"byproducts"`
	Environment map[string]interface{} `json:"environment"`
	Command     []string               `json:"command"`
}

// hashDict maps hash algorithm names to digest strings.
type hashDict struct {
	SHA256 string `json:"sha256"`
}

// signature is one entry in the signatures array.
type signature struct {
	KeyID string `json:"keyid"`
	Sig   string `json:"sig"`
}

// Parse reads an in-toto link file from r and returns a Mapped containing
// one Record representing the signed build step.
func (p *InTotoParser) Parse(r io.Reader) (parser.Mapped, error) {
	raw, err := io.ReadAll(r)
	if err != nil {
		return parser.Mapped{}, err
	}
	var lf linkFile
	if err := json.Unmarshal(raw, &lf); err != nil {
		return parser.Mapped{}, fmt.Errorf("intoto: unmarshal: %w", err)
	}
	linker := p.Linker
	if linker == "" {
		linker = "debtrace"
	}
	rec, err := buildRecord(lf, raw, linker)
	if err != nil {
		return parser.Mapped{}, err
	}
	return parser.Mapped{
		Source:       "intoto",
		NormalizedAt: time.Now().Unix(),
		Mapped:       []parser.Record{rec},
	}, nil
}

// buildRecord converts a parsed linkFile into a parser.Record.
func buildRecord(lf linkFile, raw []byte, linker string) (parser.Record, error) {
	stepHash, err := computeSignedHash(raw)
	if err != nil {
		return parser.Record{}, fmt.Errorf("intoto: compute step hash: %w", err)
	}

	stepID := fmt.Sprintf("step:linker:%s:%s@%s", linker, lf.Signed.Name, stepHash)
	command := strings.Join(lf.Signed.Command, " ")
	arch := extractArch(lf.Signed.Environment)

	step := parser.StepItem{
		ID:    stepID,
		Label: lf.Signed.Name,
		Kind:  "linker",
		Attrs: map[string]string{
			"command":      command,
			"architecture": arch,
			"name":         lf.Signed.Name,
		},
	}

	principal := buildPrincipal(lf)

	materials := buildArtifacts(lf.Signed.Name, lf.Signed.Materials, "material", "incomplete")
	products := buildArtifacts(lf.Signed.Name, lf.Signed.Products, "product", "complete")

	resource := parser.ResourceItem{
		ID:           "linker:" + linker,
		Label:        linker,
		Kind:         "linker",
		Completeness: "complete",
		Attrs:        map[string]string{"name": linker},
	}

	return parser.Record{
		Step:         step,
		Principal:    principal,
		ArtifactsIn:  materials,
		ArtifactsOut: products,
		Resources:    []parser.ResourceItem{resource},
	}, nil
}

// computeSignedHash produces a stable SHA256 of the "signed" block by
// re-marshaling through interface{} (Go sorts map keys deterministically).
func computeSignedHash(raw []byte) (string, error) {
	var outer map[string]json.RawMessage
	if err := json.Unmarshal(raw, &outer); err != nil {
		return "", err
	}
	signedRaw, ok := outer["signed"]
	if !ok {
		return "", fmt.Errorf("missing signed block")
	}
	var signedIface interface{}
	if err := json.Unmarshal(signedRaw, &signedIface); err != nil {
		return "", err
	}
	canonical, err := json.Marshal(signedIface)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(canonical)
	return hex.EncodeToString(sum[:]), nil
}

// buildPrincipal extracts the signing principal from the link's signatures.
func buildPrincipal(lf linkFile) parser.PrincipalItem {
	if len(lf.Signatures) > 0 && lf.Signatures[0].KeyID != "" {
		keyID := lf.Signatures[0].KeyID
		return parser.PrincipalItem{
			ID:    fmt.Sprintf("principal:intoto:%s", keyID),
			Label: keyID,
			Kind:  "principal",
			Attrs: map[string]string{"keyid": keyID},
		}
	}
	return parser.PrincipalItem{
		ID:    "principal:intoto:unsigned",
		Label: "unsigned",
		Kind:  "principal",
		Attrs: map[string]string{},
	}
}

// buildArtifacts converts a materials or products map into ArtifactItems.
// role is "material" or "product"; completeness is "incomplete" or "complete".
func buildArtifacts(stepName string, files map[string]hashDict, role, completeness string) []parser.ArtifactItem {
	items := make([]parser.ArtifactItem, 0, len(files))
	for filename, hdict := range files {
		id, kind := resolveArtifactID(stepName, filename, hdict.SHA256, role)
		items = append(items, parser.ArtifactItem{
			ID:           id,
			Label:        filename,
			Kind:         kind,
			Completeness: completeness,
			Attrs: map[string]string{
				"hash":     hdict.SHA256,
				"filename": filename,
			},
		})
	}
	return items
}

// resolveArtifactID returns the AStRA artifact ID and kind for a single
// material or product file.
//
// Recognition priority:
//  1. "<pkg>_<ver>_<arch>.deb"              → purl artifact, matches buildinfo + packages parsers
//  2. "<src>_<upstream>.orig.tar.<ext>"     → tarball artifact, matches buildinfo parser's ArtifactsIn
//  3. "<src>_<ver>_<arch>.buildinfo"        → buildinfo artifact, matches buildinfo parser
//  4. "<host>/<owner>/<repo>@<ref>"         → git commit artifact; sha256 field carries the commit hash
//  5. anything else                         → hash-keyed generic artifact
//
// For git refs (case 4) the sha256 field is expected to hold the git commit hash
// (SHA-1 or SHA-256, whichever the repo uses). The @<ref> suffix in the filename
// is informational and does not appear in the generated artifact ID.
func resolveArtifactID(stepName, filename, sha256hash, role string) (id, kind string) {
	if pkg, version, arch, ok := debutil.ParseDebFilename(filename); ok {
		purl := fmt.Sprintf("pkg:deb/debian/%s@%s?arch=%s", pkg, version, arch)
		return "artifact:" + purl, "deb"
	}
	if source, upstream, ok := parseDebOrigTarball(filename); ok {
		return fmt.Sprintf("artifact:tarball:deb/%s@%s", source, upstream), "tarball"
	}
	if source, version, arch, ok := parseBuildinfoFilename(filename); ok {
		return fmt.Sprintf("artifact:buildinfo:deb/%s@%s?arch=%s", source, version, arch), "buildinfo"
	}
	if repoSlug, ok := parseGitRef(filename); ok {
		// sha256hash carries the commit hash (may be 40-char SHA-1 or 64-char SHA-256)
		return fmt.Sprintf("artifact:gitcommit:%s@%s", repoSlug, sha256hash), "git-commit"
	}
	// Generic fallback keyed by SHA256 hash.
	return fmt.Sprintf("artifact:intoto:%s:%s:%s", stepName, role, sha256hash), "intoto-" + role
}

// parseDebOrigTarball recognises Debian orig tarball filenames of the form
// <source>_<upstream>.orig.tar.<ext> and returns the source and upstream version.
func parseDebOrigTarball(name string) (source, upstream string, ok bool) {
	for _, suf := range []string{".orig.tar.xz", ".orig.tar.gz", ".orig.tar.bz2", ".orig.tar.lz"} {
		if strings.HasSuffix(name, suf) {
			base := strings.TrimSuffix(name, suf)
			if i := strings.IndexByte(base, '_'); i > 0 {
				return base[:i], base[i+1:], true
			}
		}
	}
	return "", "", false
}

// parseGitRef recognises filenames of the form "<host>/<owner>/<repo>@<ref>",
// e.g. "github.com/libarchive/libarchive@v3.6.2".
// Returns the repo slug (everything before @) and ok=true when the pattern matches.
// The @<ref> suffix is stripped: the caller uses the sha256 field as the commit hash.
func parseGitRef(filename string) (repoSlug string, ok bool) {
	at := strings.LastIndex(filename, "@")
	if at < 0 {
		return "", false
	}
	slug := filename[:at] // e.g. "github.com/libarchive/libarchive"
	// Must look like a host/path: contain at least one dot (hostname) and one slash
	if !strings.Contains(slug, ".") || !strings.Contains(slug, "/") {
		return "", false
	}
	// Must not be a file path (no extension before the @)
	base := slug[strings.LastIndex(slug, "/")+1:]
	if strings.Contains(base, ".") {
		return "", false
	}
	return slug, true
}

// parseBuildinfoFilename parses a Debian .buildinfo filename of the form
//
//	<source>_<version>_<arch>.buildinfo
//
// Returns ok=false for any other filename form.
//
// Example:
//
//	libarchive_3.6.2-1_amd64.buildinfo → ("libarchive", "3.6.2-1", "amd64", true)
func parseBuildinfoFilename(name string) (source, version, arch string, ok bool) {
	if !strings.HasSuffix(name, ".buildinfo") {
		return "", "", "", false
	}
	parts := strings.SplitN(strings.TrimSuffix(name, ".buildinfo"), "_", 3)
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return "", "", "", false
	}
	return parts[0], parts[1], parts[2], true
}

// extractArch attempts to find a build architecture from the in-toto
// environment's "variables" list.  It checks common Debian/environment
// variable names and returns the value of the first match found.
func extractArch(env map[string]interface{}) string {
	vars, ok := env["variables"]
	if !ok {
		return ""
	}
	list, ok := vars.([]interface{})
	if !ok {
		return ""
	}
	prefixes := []string{"ARCH=", "DEB_BUILD_ARCH=", "DEB_HOST_ARCH="}
	for _, v := range list {
		s, ok := v.(string)
		if !ok {
			continue
		}
		for _, prefix := range prefixes {
			if strings.HasPrefix(s, prefix) {
				return strings.TrimPrefix(s, prefix)
			}
		}
	}
	return ""
}
