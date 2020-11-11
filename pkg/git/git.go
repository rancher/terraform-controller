package git

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
)

func GetCommit(ctx context.Context, url, branch, tag string, auth *Auth) (string, error) {
	url, env, close := auth.Populate(url)
	defer close()

	lines, err := git(ctx, env, "ls-remote", url, formatRef(branch, tag))
	if err != nil {
		return "", err
	}

	return firstField(lines, fmt.Sprintf("no commit for branch: %s or tag: %s", branch, tag))
}

func CloneRepo(ctx context.Context, url string, commit string, auth *Auth) error {
	url, env, close := auth.Populate(url)
	defer close()

	lines, err := git(ctx, env, "clone", "-n", url, ".")
	if err != nil {
		return err
	}

	logrus.Infof("Output from git clone %v", lines)

	lines, err = git(ctx, env, "checkout", commit)
	if err != nil {
		return err
	}

	logrus.Infof("Output from git checkout %v", lines)

	return nil
}
