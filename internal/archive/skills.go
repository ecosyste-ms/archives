package archive

import (
	"errors"
	"log/slog"
	"math"

	"github.com/git-pkgs/skills"
)

type SkillsResult struct {
	Skills []SkillResult `json:"skills"`
	Errors []string      `json:"errors"`
}

type SkillResult struct {
	Path          string         `json:"path"`
	Name          string         `json:"name"`
	Description   string         `json:"description"`
	License       string         `json:"license"`
	Compatibility string         `json:"compatibility"`
	AllowedTools  string         `json:"allowed_tools"`
	Metadata      map[string]any `json:"metadata"`
	Extra         map[string]any `json:"extra"`
	Body          string         `json:"body"`
	Files         []string       `json:"files"`
	HasScripts    bool           `json:"has_scripts"`
	SchemaJSON    string         `json:"schema_json"`
	SourceHash    string         `json:"source_hash"`
	Warnings      []string       `json:"warnings"`
}

// Skills returns the agent skills (SKILL.md directories) in the archive.
// A SKILL.md that fails to parse is reported in Errors without hiding the
// others.
func (a *RemoteArchive) Skills() (*SkillsResult, error) {
	result := &SkillsResult{Skills: []SkillResult{}, Errors: []string{}}
	b, err := a.browse()
	if errors.Is(err, ErrInvalidArchive) {
		slog.Info("archive skills failed", "error", err)
		return result, nil
	}
	if err != nil {
		return nil, err
	}
	if b == nil {
		return result, nil
	}
	defer b.close()

	// The index is already in memory and bounded by maxFileCount, so search
	// every depth rather than the on-disk default.
	found, err := skills.Discover(newBrowserFS(b), skills.WithMaxDepth(math.MaxInt))
	for _, s := range found {
		result.Skills = append(result.Skills, newSkillResult(s))
	}
	if err != nil {
		if b.ctx.Err() != nil {
			return nil, b.ctx.Err()
		}
		for _, e := range unwrapJoined(err) {
			result.Errors = append(result.Errors, e.Error())
		}
	}
	return result, nil
}

func newSkillResult(s *skills.Skill) SkillResult {
	files := s.Files
	if files == nil {
		files = []string{}
	}
	warnings := s.Warnings
	if warnings == nil {
		warnings = []string{}
	}
	return SkillResult{
		Path:          s.Dir,
		Name:          s.Name,
		Description:   s.Description,
		License:       s.License,
		Compatibility: s.Compatibility,
		AllowedTools:  s.AllowedTools,
		Metadata:      s.Metadata,
		Extra:         s.Extra,
		Body:          s.Body,
		Files:         files,
		HasScripts:    s.HasScripts(),
		SchemaJSON:    s.SchemaJSON,
		SourceHash:    s.SourceHash,
		Warnings:      warnings,
	}
}

func unwrapJoined(err error) []error {
	if joined, ok := err.(interface{ Unwrap() []error }); ok {
		return joined.Unwrap()
	}
	return []error{err}
}
