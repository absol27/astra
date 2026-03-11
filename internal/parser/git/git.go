/*
Package git implements the GitParser for AStRA.

GitParser parses Git repositories and maps commits and files
into AStRA’s DAG-based artifact graph.

For each commit:
- The commit is represented as a step.
- Authors are represented as principals.
- The Git is treated as a resource.
- Files are tracked as input/output artifacts.
- Parent commits are mapped as input artifacts.
- The commit itself and changed files are mapped as output artifacts.
*/
package git

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	parser "github.com/abuishgair/astra/internal/parser"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/utils/merkletrie"
)

// GitParser implements parser.Parser for Git repositories.
type GitParser struct{}

// MakeArtifactID returns a namespaced AStRA artifact ID for a file
// at a specific commit in the given repository.
func MakeArtifactID(repoURL string, commitHash, filePath string) string {
	repoSlug := getRepoSlug(repoURL)
	return fmt.Sprintf("artifact:gitfile:%s@%s:%s", repoSlug, commitHash, filePath)
}

/*
MakeStepID returns namespaced AStRA Step ID for a Git commit.
Step ID format: step:commit:<host>/<owner>/<repo>@<commit-hash>
*/
func MakeStepID(repoURL string, commitHash string) string {
	repoSlug := getRepoSlug(repoURL)
	return fmt.Sprintf("step:commit:%s@%s", repoSlug, commitHash)
}

// MakeCommitArtifactID returns namespaced AStRA artifact ID
// for a Git commit.
// Commit artifact ID format: artifact:gitcommit:<host>/<owner>/<repo>@<commit-hash>
func MakeCommitArtifactID(repoURL string, commitHash string) string {
	repoSlug := getRepoSlug(repoURL)
	return fmt.Sprintf("artifact:gitcommit:%s@%s", repoSlug, commitHash)
}

// getRepoSlug normalizes a Git repository URL into a host/owner/repo slug.
// Supported URL formats include:
// https://github.com/owner/repo.git
// https://github.com/owner/repo
// git@github.com:owner/repo.git
func getRepoSlug(raw string) string {
	// Handle SSH scp-style: git@github.com:owner/repo.git
	if strings.HasPrefix(raw, "git@") {
		parts := strings.SplitN(raw, ":", 2)
		if len(parts) == 2 {
			host := strings.TrimPrefix(parts[0], "git@")
			path := strings.TrimSuffix(parts[1], ".git")
			return host + "/" + path
		}
		return raw
	}

	// Handle real URLs: https://..., ssh://...
	u, err := url.Parse(raw)
	if err == nil && u.Host != "" {
		path := strings.TrimPrefix(u.Path, "/")
		path = strings.TrimSuffix(path, ".git")
		return u.Host + "/" + path
	}

	return raw
}

// GetCommitIO computes the input and output files for a commit in a Git repository.
// - Inputs: files from parent commits that are modified or deleted.
// - Outputs: files added or modified in this commit.
// - Returns slices of *object.File for inputs and outputs.
func GetCommitIO(repo *git.Repository, hash string) (inputs []*object.File, outputs []*object.File, err error) {
	commit, err := repo.CommitObject(plumbing.NewHash(hash))
	if err != nil {
		return nil, nil, err
	}

	currTree, err := commit.Tree()
	if err != nil {
		return nil, nil, err
	}

	var parentTree *object.Tree
	var parentHash plumbing.Hash
	if commit.NumParents() > 0 {
		parent, err := commit.Parent(0)
		if err != nil {
			return nil, nil, err
		}
		parentHash = parent.Hash
		parentTree, err = parent.Tree()
		if err != nil {
			return nil, nil, err
		}
	}

	seenIn := map[string]bool{}
	seenOut := map[string]bool{}

	// if this is a root commit: everything is output
	if parentTree == nil {
		if err := currTree.Files().ForEach(func(f *object.File) error {
			if seenOut[f.Name] {
				return nil
			}
			seenOut[f.Name] = true
			outputs = append(outputs, f)
			return nil
		}); err != nil {
			return nil, nil, err
		}
		return inputs, outputs, nil
	}

	changes, err := object.DiffTree(parentTree, currTree)
	if err != nil {
		return nil, nil, err
	}

	for _, ch := range changes {
		action, _ := ch.Action()

		switch action {
		case merkletrie.Insert:
			path := ch.To.Name
			if seenOut[path] {
				continue
			}
			f, err := currTree.File(path)
			if err != nil {
				// submodule/symlink/rename edge-case; can't fetch file blob
				// skip metadata but continue
				continue
			}
			seenOut[path] = true
			outputs = append(outputs, f)

		case merkletrie.Delete:
			path := ch.From.Name
			if seenIn[path] {
				continue
			}
			f, err := parentTree.File(path)
			if err != nil {
				return nil, nil, err
			}
			seenIn[path] = true
			inputs = append(inputs, f)

		case merkletrie.Modify:
			inPath := ch.From.Name
			outPath := ch.To.Name

			if !seenIn[inPath] {
				fBefore, err := parentTree.File(inPath)
				if err != nil {
					return nil, nil, err
				}
				seenIn[inPath] = true
				inputs = append(inputs, fBefore)
			}

			if !seenOut[outPath] {
				fAfter, err := currTree.File(outPath)
				if err != nil {
					return nil, nil, err
				}
				seenOut[outPath] = true
				outputs = append(outputs, fAfter)
			}
		}
	}

	_ = parentHash // keep to later build artifact IDs for inputs at parent commit
	return inputs, outputs, nil
}

// Parse clones the Git repository from the given URL, extracts commits,
// and maps them into a parser Mapped structure for AStRA.
//
// Each commit is represented as a step, authors as principals, Git as a resource
// parent commits as input artifacts, and the commit itself plus changed files
// as output artifacts.
func (p *GitParser) Parse(repoURL string) (parser.Mapped, error) {
	tmpDir := filepath.Join(os.TempDir(), "gitrepo-history")
	_ = os.RemoveAll(tmpDir) // cleanup from previous runs

	fmt.Println("Cloning:", repoURL)
	fmt.Println("Into   :", tmpDir)

	repo, err := git.PlainClone(tmpDir, false, &git.CloneOptions{
		URL:      repoURL,
		Progress: os.Stdout,
	})
	if err != nil {
		return parser.Mapped{}, fmt.Errorf("clone error: %w", err)
	}

	rem, err := repo.Remote("origin")
	if err != nil {
		return parser.Mapped{}, fmt.Errorf("remote error: %w", err)
	}
	remoteURLs := rem.Config().URLs
	if len(remoteURLs) == 0 {
		return parser.Mapped{}, fmt.Errorf("origin remote has no URLs")
	}
	remoteURL := remoteURLs[0]
	fmt.Println("remote:", remoteURL)

	ref, err := repo.Head()
	if err != nil {
		return parser.Mapped{}, fmt.Errorf("head error: %w", err)
	}

	commits, err := repo.Log(&git.LogOptions{From: ref.Hash()})
	if err != nil {
		return parser.Mapped{}, fmt.Errorf("log error: %w", err)
	}

	out := parser.Mapped{
		Source:       "go-git",
		NormalizedAt: time.Now().Unix(),
	}

	err = commits.ForEach(func(c *object.Commit) error {
		rec := parser.Record{
			Step: parser.Item{
				ID:    MakeStepID(remoteURL, c.Hash.String()),
				Label: "Commit",
				Kind:  "step",
				Attrs: map[string]string{
					"phase":   "source",
					"message": strings.TrimSpace(c.Message),
				},
			},
			Principal: parser.Item{
				ID:    "principal:" + c.Author.Email,
				Label: c.Author.Name,
				Kind:  "principal",
				Attrs: map[string]string{"email": c.Author.Email},
			},
		}

		// Compute IO as file objects
		inputs, outputs, err := GetCommitIO(repo, c.Hash.String())
		if err != nil {
			return err
		}

		// Parent hash is needed to version the "before" (input) artifacts
		parentHash := ""
		// Parent commit(s) as input artifacts
		if c.NumParents() > 0 {
			for i := 0; i < c.NumParents(); i++ {
				parent, err := c.Parent(i)
				if err != nil {
					return err
				}
				parentHash = parent.Hash.String()
				rec.ArtifactsIn = append(rec.ArtifactsIn, parser.Item{
					ID:    MakeCommitArtifactID(remoteURL, parentHash),
					Label: parentHash,
					Kind:  "git-commit",
					Attrs: map[string]string{
						"role":  "parent",
						"index": strconv.Itoa(i),
					},
				})
			}
		}

		// ArtifactsIn (before versions)
		// Root commit => parentHash == "" and inputs should be empty.
		// input files are connected to parent 0 only

		for _, f := range inputs {
			if f == nil || parentHash == "" {
				continue
			}
			rec.ArtifactsIn = append(rec.ArtifactsIn, parser.Item{
				ID:    MakeArtifactID(remoteURL, parentHash, f.Name),
				Label: f.Name,
				Kind:  "git-file",
				Attrs: map[string]string{
					"hash": f.Hash.String(),
					"size": strconv.FormatInt(f.Size, 10),
					"mode": f.Mode.String(),
				},
			})
		}

		//add the commit as output artifact
		rec.ArtifactsOut = append(rec.ArtifactsOut, parser.Item{
			ID:    MakeCommitArtifactID(remoteURL, c.Hash.String()),
			Label: c.Hash.String(),
			Kind:  "git-commit",
			Attrs: map[string]string{
				"message": strings.TrimSpace(c.Message),
				"author":  c.Author.Email,
				"time":    strconv.FormatInt(c.Author.When.Unix(), 10), //TODO check format consistency
			},
		})

		// ArtifactsOut (after versions)
		for _, f := range outputs {
			if f == nil {
				continue
			}
			rec.ArtifactsOut = append(rec.ArtifactsOut, parser.Item{
				ID:    MakeArtifactID(remoteURL, c.Hash.String(), f.Name),
				Label: f.Name,
				Kind:  "git-file",
				Attrs: map[string]string{
					"content-hash": f.Hash.String(),
					"size":         strconv.FormatInt(f.Size, 10),
					"mode":         f.Mode.String(),
				},
			})
		}

		// Resources
		rec.Resources = append(rec.Resources, parser.Item{
			ID:    "resource:git",
			Label: "git",
			Kind:  "vcs",
		})

		out.Mapped = append(out.Mapped, rec)
		return nil
	})
	if err != nil {
		return parser.Mapped{}, err
	}

	return out, nil
}
