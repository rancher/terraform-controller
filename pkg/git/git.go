package git

import (
	"context"
	"fmt"
)

func BranchCommit(ctx context.Context, url string, branch string, auth *Auth) (string, error) {
	url, env, close := auth.Populate(url)
	defer close()

	lines, err := git(ctx, env, "ls-remote", url, formatRefForBranch(branch))
	if err != nil {
		return "", err
	}

	return firstField(lines, fmt.Sprintf("no commit for branch: %s", branch))
}
