package archive

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

const repomixWaitDelay = 5 * time.Second

// Variables so tests can shrink them.
var (
	repomixTimeout    = 60 * time.Second
	maxRepopackOutput = int64(maxFileSize)
)

type RepopackResult struct {
	Output string `json:"output"`
}

func (a *RemoteArchive) Repopack() (*RepopackResult, error) {
	dir, err := os.MkdirTemp("", "archives-*")
	if err != nil {
		return nil, err
	}
	defer cleanupDirectory(dir)

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

	// Keep the config outside the input tree and bypass archive-supplied configs.
	configPath := filepath.Join(dir, "repomix.config.json")
	if err := os.WriteFile(configPath, []byte("{}"), downloadFileMode); err != nil {
		return nil, fmt.Errorf("writing repomix config: %w", err)
	}

	ctx, cancel := context.WithTimeout(a.context, repomixTimeout)
	defer cancel()

	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "repomix", ".", "--output", "repomix-output.txt", "--verbose", "--config", configPath)
	cmd.Dir = extractDir
	cmd.Stderr = &stderr
	cmd.WaitDelay = repomixWaitDelay
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			slog.Info("repomix aborted", "error", ctx.Err())
			return nil, nil
		}
		return nil, fmt.Errorf("repomix failed: %w: %s", err, stderr.String())
	}

	data, err := readRepopackOutput(filepath.Join(extractDir, "repomix-output.txt"))
	if err != nil {
		return nil, err
	}

	return &RepopackResult{
		Output: string(data),
	}, nil
}

func readRepopackOutput(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("reading repomix output: %w", err)
	}
	defer func() { _ = f.Close() }()

	data, err := io.ReadAll(io.LimitReader(f, maxRepopackOutput+1))
	if err != nil {
		return nil, fmt.Errorf("reading repomix output: %w", err)
	}
	if int64(len(data)) > maxRepopackOutput {
		return nil, fmt.Errorf("%w: repomix output is larger than %d bytes", ErrTooLarge, maxRepopackOutput)
	}
	return data, nil
}
