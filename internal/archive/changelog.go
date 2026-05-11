package archive

import (
	"github.com/git-pkgs/changelog"
)

func parseChangelogContent(content string) map[string]string {
	p := changelog.Parse(content)
	result := make(map[string]string)
	for _, version := range p.Versions() {
		entry, ok := p.Entry(version)
		if ok {
			result[version] = entry.Content
		}
	}
	return result
}
