package utils

import (
	"fmt"
	"regexp"
)

// tagNameRegexp captures the repository name (owner and ID) and tag ID from a
// tag resource name.
var tagNameRegexp = regexp.MustCompile(`repositories/(([^/]+)/([^/]+))/tags/([^/]+)`)

// RepositoryTagName is the resource name of a repositoryTag.
type RepositoryTagName string

// ExtractRepositoryAndID breaks a RepositoryTagName down into a repository ID
// and a tag ID. If the name is invalid, an error is returned.
func (t RepositoryTagName) ExtractRepositoryAndID() (repo, id string, err error) {
	matches := tagNameRegexp.FindStringSubmatch(string(t))
	if len(matches) == 0 {
		err = fmt.Errorf("invalid tag name")
		return
	}

	return matches[1], matches[4], nil
}

const rawRepositoryName = "repositories/%s/tags/%s"

// NewRepositoryTagName composes a tag name from its parent and ID.
func NewRepositoryTagName(repo, id string) RepositoryTagName {
	name := fmt.Sprintf(rawRepositoryName, repo, id)
	return RepositoryTagName(name)
}
