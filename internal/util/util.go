package util

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"

	"github.com/gernest/front"
	"github.com/instill-ai/model-backend/pkg/datamodel"
	"github.com/mitchellh/mapstructure"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/config"
	"gopkg.in/src-d/go-git.v4/plumbing"
)

func IsGitHubURL(input string) bool {
	if input == "" {
		return false
	}
	u, err := url.Parse(input)
	if err != nil {
		return false
	}
	host := u.Host
	if strings.Contains(host, ":") {
		host, _, err = net.SplitHostPort(host)
		if err != nil {
			return false
		}
	}
	return host == "github.com"
}

type ModelMeta struct {
	Tags []string
	Task string
}

func GetModelMetaFromReadme(readmeFilePath string) (*ModelMeta, error) {
	if _, err := os.Stat(readmeFilePath); err != nil {
		return &ModelMeta{}, err
	}
	file, err := os.Open(readmeFilePath)
	if err != nil {
		return &ModelMeta{}, err
	}
	fm := front.NewMatter()
	fm.Handle("---", front.YAMLHandler)
	meta, _, err := fm.Parse(file)
	if err != nil {
		return &ModelMeta{}, err
	}
	var modelMeta ModelMeta
	err = mapstructure.Decode(meta, &modelMeta)

	return &modelMeta, err
}

func GitHubClone(dir string, github datamodel.GitHub) error {
	if !IsGitHubURL(github.RepoUrl) {
		return fmt.Errorf("Invalid GitHub URL")
	}

	r, err := git.PlainClone(dir, false, &git.CloneOptions{
		URL: github.RepoUrl,
	})
	if err != nil {
		return err
	}
	w, err := r.Worktree()
	if err != nil {
		return err
	}
	err = r.Fetch(&git.FetchOptions{
		RefSpecs: []config.RefSpec{"refs/*:refs/*", "HEAD:refs/heads/HEAD"},
	})
	if err != nil {
		return err
	}
	if github.GitRef.Branch != "" {
		err = w.Checkout(&git.CheckoutOptions{
			Branch: plumbing.ReferenceName(fmt.Sprintf("refs/heads/%s", github.GitRef.Branch)),
			Force:  true,
		})
	} else if github.GitRef.Tag != "" {
		err = w.Checkout(&git.CheckoutOptions{
			Branch: plumbing.ReferenceName(fmt.Sprintf("refs/tags/%s", github.GitRef.Tag)),
			Force:  true,
		})
	} else if github.GitRef.Commit != "" {
		err = w.Checkout(&git.CheckoutOptions{
			Hash:  plumbing.NewHash(github.GitRef.Commit),
			Force: true,
		})
	} else {
		err = w.Checkout(&git.CheckoutOptions{
			Branch: plumbing.ReferenceName("refs/heads/main"), // default is main branch
			Force:  true,
		})
	}

	return err
}
