package core

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/utils/merkletrie"
)

const (
	EmptyTreeCommitId = "4b825dc642cb6eb9a060e54bf8d69288fbee4904"
)

func CloneRepository(url *string, branch *string, depth int) (*git.Repository, string, error) {
	urlVal := *url
	branchVal := *branch

	// Create temp directory with a more specific prefix
	dir, err := ioutil.TempDir("", fmt.Sprintf("gitrob_repo_%s_", filepath.Base(urlVal)))
	if err != nil {
		return nil, "", fmt.Errorf("failed to create temp directory: %v", err)
	}

	repository, err := git.PlainClone(dir, false, &git.CloneOptions{
		URL:           urlVal,
		Depth:         depth,
		ReferenceName: plumbing.ReferenceName(fmt.Sprintf("refs/heads/%s", branchVal)),
		SingleBranch:  true,
		Tags:          git.NoTags,
	})

	// If clone fails, clean up the directory
	if err != nil {
		os.RemoveAll(dir)
		if err.Error() == "remote repository is empty" {
			return nil, "", err
		}
		return nil, "", fmt.Errorf("failed to clone repository: %v", err)
	}

	return repository, dir, nil
}

func GetRepositoryHistory(repository *git.Repository) ([]*object.Commit, error) {
	var commits []*object.Commit
	ref, err := repository.Head()
	if err != nil {
		return nil, err
	}
	cIter, err := repository.Log(&git.LogOptions{From: ref.Hash()})
	if err != nil {
		return nil, err
	}
	cIter.ForEach(func(c *object.Commit) error {
		commits = append(commits, c)
		return nil
	})
	return commits, nil
}

func GetChanges(commit *object.Commit, repo *git.Repository) (object.Changes, error) {
	parentCommit, err := GetParentCommit(commit, repo)
	if err != nil {
		return nil, err
	}

	commitTree, err := commit.Tree()
	if err != nil {
		return nil, err
	}

	parentCommitTree, err := parentCommit.Tree()
	if err != nil {
		return nil, err
	}

	changes, err := object.DiffTree(parentCommitTree, commitTree)
	if err != nil {
		return nil, err
	}
	return changes, nil
}

func GetParentCommit(commit *object.Commit, repo *git.Repository) (*object.Commit, error) {
	if commit.NumParents() == 0 {
		parentCommit, err := repo.CommitObject(plumbing.NewHash(EmptyTreeCommitId))
		if err != nil {
			return nil, err
		}
		return parentCommit, nil
	}
	parentCommit, err := commit.Parents().Next()
	if err != nil {
		return nil, err
	}
	return parentCommit, nil
}

func GetChangeAction(change *object.Change) string {
	action, err := change.Action()
	if err != nil {
		return "Unknown"
	}
	switch action {
	case merkletrie.Insert:
		return "Insert"
	case merkletrie.Modify:
		return "Modify"
	case merkletrie.Delete:
		return "Delete"
	default:
		return "Unknown"
	}
}

func GetChangeContent(change *object.Change) ([]byte, error) {
	action := GetChangeAction(change)
	if action == "Delete" {
		return nil, nil
	}

	// Skip SVG files and other potentially problematic formats
	path := GetChangePath(change)
	if strings.HasSuffix(strings.ToLower(path), ".svg") {
		return nil, nil
	}

	patch, err := change.Patch()
	if err != nil {
		return nil, err
	}

	var content strings.Builder
	for _, filePatch := range patch.FilePatches() {
		if filePatch.IsBinary() {
			continue
		}

		// Skip if total content size is too large (>1MB)
		var totalSize int
		for _, chunk := range filePatch.Chunks() {
			totalSize += len(chunk.Content())
			if totalSize > 1024*1024 {
				return nil, nil
			}
		}

		for _, chunk := range filePatch.Chunks() {
			content.WriteString(chunk.Content())
		}
	}

	return []byte(content.String()), nil
}

func GetChangePath(change *object.Change) string {
	action := GetChangeAction(change)
	if action == "Delete" {
		return change.From.Name
	}
	return change.To.Name
}
