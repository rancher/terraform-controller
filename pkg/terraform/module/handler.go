package module

import (
	"context"
	"time"

	"github.com/pkg/errors"
	v1 "github.com/rancher/terraform-controller/pkg/apis/terraformcontroller.cattle.io/v1"
	"github.com/rancher/terraform-controller/pkg/digest"
	tfv1 "github.com/rancher/terraform-controller/pkg/generated/controllers/terraformcontroller.cattle.io/v1"
	"github.com/rancher/terraform-controller/pkg/git"
	"github.com/rancher/terraform-controller/pkg/interval"
	corev1 "github.com/rancher/wrangler/pkg/generated/controllers/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewHandler(ctx context.Context, modules tfv1.ModuleController, secrets corev1.SecretController) *Handler {
	return &Handler{
		ctx:     ctx,
		modules: modules,
		secrets: secrets,
	}
}

type Handler struct {
	ctx     context.Context
	modules tfv1.ModuleController
	secrets corev1.SecretController
}

func (h *Handler) OnChange(key string, module *v1.Module) (*v1.Module, error) {
	if module == nil {
		return nil, nil
	}
	if module.Spec.Git.IntervalSeconds == 0 {
		module.Spec.Git.IntervalSeconds = int(interval.DefaultInterval / time.Second)
	}

	if isPolling(module.Spec) && needsUpdate(module) {
		return h.updateCommit(key, module)
	}
	hash := computeHash(module)
	if module.Status.ContentHash != hash {
		return h.updateHash(module, hash)
	}

	h.modules.EnqueueAfter(module.Namespace, module.Name, time.Duration(module.Spec.Git.IntervalSeconds)*time.Second)

	return h.modules.Update(module)
}

func (h *Handler) OnRemove(key string, module *v1.Module) (*v1.Module, error) {
	//nothing to do here
	return module, nil
}

func (h *Handler) updateHash(module *v1.Module, hash string) (*v1.Module, error) {
	module = module.DeepCopy()
	module.Status.Content = module.Spec.ModuleContent
	module.Status.ContentHash = hash
	if isPolling(module.Spec) && module.Status.GitChecked != nil {
		module.Status.Content.Git.Commit = module.Status.GitChecked.Commit
	}
	return h.modules.Update(module)
}

func (h *Handler) updateCommit(key string, module *v1.Module) (*v1.Module, error) {
	branch := module.Spec.Git.Branch
	tag := module.Spec.Git.Tag

	if branch == "" {
		branch = "master"
	}
	// unset branch if tag is set
	if tag != "" {
		branch = ""
	}

	auth, err := h.getAuth(module.Namespace, module.Spec)
	if err != nil {
		return nil, err
	}

	commit, err := git.GetCommit(h.ctx, module.Spec.Git.URL, branch, tag, &auth)
	if err != nil {
		return nil, err
	}

	gitChecked := module.Spec.Git
	gitChecked.Commit = commit
	module.Status.GitChecked = &gitChecked
	module.Status.CheckTime = metav1.Now()

	v1.ModuleConditionGitUpdated.True(module)

	return h.modules.Update(module)
}

func (h *Handler) getAuth(ns string, spec v1.ModuleSpec) (git.Auth, error) {
	auth := git.Auth{}
	name := spec.Git.SecretName

	if name == "" {
		return auth, nil
	}

	secret, err := h.secrets.Get(ns, name, metav1.GetOptions{})
	if err != nil {
		return auth, errors.Wrapf(err, "fetch git secret %s:", name)
	}

	return git.FromSecret(secret.Data)
}

func needsUpdate(m *v1.Module) bool {
	return interval.NeedsUpdate(m.Status.CheckTime.Time, time.Duration(m.Spec.Git.IntervalSeconds)*time.Second) ||
		v1.ModuleConditionGitUpdated.IsFalse(m) ||
		m.Status.GitChecked == nil ||
		m.Status.GitChecked.URL != m.Spec.Git.URL ||
		m.Status.GitChecked.Branch != m.Spec.Git.Branch
}

func isPolling(spec v1.ModuleSpec) bool {
	return len(spec.Content) == 0 &&
		spec.Git.URL != "" &&
		spec.Git.Commit == ""
}

func computeHash(obj *v1.Module) string {
	if len(obj.Spec.Content) > 0 {
		return digest.SHA256Map(obj.Spec.Content)
	}

	git := obj.Spec.Git
	if git.URL == "" {
		return ""
	}

	if isPolling(obj.Spec) && obj.Status.GitChecked != nil {
		git.Commit = obj.Status.GitChecked.Commit
	}

	if git.Commit != "" {
		return digest.SHA256Map(map[string]string{
			"url":    git.URL,
			"commit": git.Commit,
		})
	}

	if git.Tag != "" {
		return digest.SHA256Map(map[string]string{
			"url": git.URL,
			"tag": git.Tag,
		})
	}

	return ""
}
