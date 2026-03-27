package archive

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type RepopackResult struct {
	Output string `json:"output"`
}

func (a *RemoteArchive) Repopack() (*RepopackResult, error) {
	dir, err := os.MkdirTemp("", "archives-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)

	if err := a.Download(dir); err != nil {
		return nil, err
	}

	extractDir, err := a.Extract(dir)
	if err != nil {
		return nil, err
	}
	if extractDir == "" {
		return nil, nil
	}

	cmd := exec.Command("repomix", ".", "--output", "repomix-output.txt", "--verbose")
	cmd.Dir = extractDir
	cmd.Run()

	outputPath := filepath.Join(extractDir, "repomix-output.txt")
	data, err := os.ReadFile(outputPath)
	if err != nil {
		return nil, fmt.Errorf("reading repomix output: %w", err)
	}

	return &RepopackResult{
		Output: string(data),
	}, nil
}
