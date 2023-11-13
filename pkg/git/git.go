package git

import (
	"fmt"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"path/filepath"
	"strings"
)

func CloneRepo(basedir, repo, username, password string) error {
	p := filepath.Join(basedir, repo)
	if strings.Contains(p, "..") {
		return fmt.Errorf("%s seems like a bogus directory", p)
	}
	_, err := git.PlainClone(p, false, &git.CloneOptions{
		URL: fmt.Sprintf("https://github.com/%s", repo),
		Auth: &http.BasicAuth{
			Username: username,
			Password: password,
		},
		Depth: 1,
	})
	return err
}
