/*
Package buildinfo implements the BuildinfoParser for AStRA.

BuildinfoParser parses Debian .buildinfo files and maps the build event
into AStRA's record structure for the mapper.

For a buildinfo file:
- The dpkg-buildpackage invocation is represented as a step.
- The build origin (e.g. "Debian") is the principal; the PGP key ID is stored in its attrs.
- The upstream source tarball is the resource that carries out the step.
- Installed build dependencies are input artifacts (consumed by the step).
- Output .deb files are output artifacts (produced by the step).

Dependency identifiers use Package URL (purl) format:
pkg:deb/debian/<name>@<version>
*/
package buildinfo

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/ProtonMail/gopenpgp/v3/crypto"
	parser "github.com/TSELab/astra/internal/parser"
)

type BuildinfoParser struct{}

func (p *BuildinfoParser) Parse(path string) (parser.Mapped, error) {
	return parseBuildinfo(path)
}

func parseBuildinfo(path string) (parser.Mapped, error) {
	file, err := os.Open(path)
	if err != nil {
		return parser.Mapped{}, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	var source, version, buildDate, buildOrigin, buildArch string
	var outputItems []parser.Item
	var depItems []parser.Item
	var pgpLines []string
	seenDeps := map[string]bool{}

	outputSection, dependsSection, pgpSection := false, false, false
	re := regexp.MustCompile(`([a-zA-Z0-9.+:~\-]+) \(= ([^\)]+)\)`)

	for scanner.Scan() {
		line := scanner.Text()

		switch {
		case strings.HasPrefix(line, "Source:"):
			source = strings.TrimSpace(strings.TrimPrefix(line, "Source:"))
		case strings.HasPrefix(line, "Version:"):
			version = strings.TrimSpace(strings.TrimPrefix(line, "Version:"))
		case strings.HasPrefix(line, "Build-Architecture:"):
			buildArch = strings.TrimSpace(strings.TrimPrefix(line, "Build-Architecture:"))
		case strings.HasPrefix(line, "Build-Date:"):
			buildDate = strings.TrimSpace(strings.TrimPrefix(line, "Build-Date:"))
		case strings.HasPrefix(line, "Build-Origin:"):
			buildOrigin = strings.TrimSpace(strings.TrimPrefix(line, "Build-Origin:"))
		case strings.HasPrefix(line, "Checksums-Sha256:"):
			outputSection = true
			continue
		case strings.HasPrefix(line, "Installed-Build-Depends:"):
			dependsSection = true
			continue
		case strings.HasPrefix(line, "-----BEGIN PGP SIGNATURE-----"):
			pgpSection = true
		case strings.HasPrefix(line, "-----END PGP SIGNATURE-----"):
			pgpSection = false
			continue
		case outputSection && strings.TrimSpace(line) == "":
			outputSection = false
		case dependsSection && strings.TrimSpace(line) == "":
			dependsSection = false
		}

		if pgpSection {
			pgpLines = append(pgpLines, line)
			continue
		}

		if outputSection && strings.HasSuffix(strings.TrimSpace(line), ".deb") {
			parts := strings.Fields(line)
			if len(parts) == 3 {
				hash, size, filename := parts[0], parts[1], parts[2]
				pkgName := strings.SplitN(filename, "_", 2)[0]
				purl := fmt.Sprintf("pkg:deb/debian/%s@%s?arch=%s", pkgName, version, buildArch)
				outputItems = append(outputItems, parser.Item{
					ID:    purl,
					Label: filename,
					Kind:  "deb",
					Attrs: map[string]string{
						"purl":     purl,
						"hash":     hash,
						"size":     size,
						"filename": filename,
						"version":  version,
					},
				})
			}
		} else if dependsSection && strings.TrimSpace(line) != "" {
			matches := re.FindAllStringSubmatch(line, -1)
			for _, match := range matches {
				pkg := strings.TrimSpace(match[1])
				ver := strings.TrimSpace(match[2])
				purl := fmt.Sprintf("pkg:deb/debian/%s@%s", pkg, ver)
				if !seenDeps[purl] {
					seenDeps[purl] = true
					depItems = append(depItems, parser.Item{
						ID:    purl,
						Label: pkg,
						Kind:  "deb",
						Attrs: map[string]string{
							"purl":    purl,
							"version": ver,
						},
					})
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return parser.Mapped{}, err
	}

	// Extract PGP signing key ID from the signature block.
	var keyID string
	if len(pgpLines) > 0 {
		pgpText := strings.Join(pgpLines, "\n")
		if pgpMsg, err := crypto.NewPGPMessageFromArmored(pgpText); err == nil {
			if ids, ok := pgpMsg.HexSignatureKeyIDs(); ok && len(ids) > 0 {
				keyID = ids[0]
			}
		}
	}

	// TODO(resource semantics): in the model, a resource "carries out" a step, which fits a tool like dpkg-buildpackage.
	// So that would mean, the source tarball is arguably an input consumed by the step, not the agent that executes it?
	upstreamVersion := strings.SplitN(version, "-", 2)[0]
	tarball := fmt.Sprintf("%s_%s.orig.tar.xz", source, upstreamVersion)
	// not sure what the conventional purl format for source tarballs is, since they don't have a version in the same way
	// as packages do. Using the upstream version for now.
	tarballPURL := fmt.Sprintf("pkg:deb/debian/%s@%s?arch=source", source, upstreamVersion)
	tarballResource := parser.Item{
		ID:    tarballPURL,
		Label: tarball,
		Kind:  "tarball",
		Attrs: map[string]string{
			"purl":   tarballPURL,
			"format": "orig.tar.xz",
		},
	}

	principalAttrs := map[string]string{}
	if keyID != "" {
		principalAttrs["pgp_key_id"] = keyID
	}
	principal := parser.Item{
		ID:    fmt.Sprintf("principal:%s", buildOrigin),
		Label: "Debian Build Infrastructure",
		Kind:  "principal",
		Attrs: principalAttrs,
	}

	rec := parser.Record{
		Step: parser.Item{
			ID:    fmt.Sprintf("step:build:deb/%s@%s", source, version),
			Label: "dpkg-buildpackage",
			Kind:  "build",
			Attrs: map[string]string{
				"command":      "dpkg-buildpackage",
				"timestamp":    buildDate,
				"architecture": buildArch,
			},
		},
		Principal:    principal,
		ArtifactsIn:  depItems,
		ArtifactsOut: outputItems,
		Resources:    []parser.Item{tarballResource},
	}

	return parser.Mapped{
		Source:       "buildinfo",
		NormalizedAt: time.Now().Unix(),
		Mapped:       []parser.Record{rec},
	}, nil
}
