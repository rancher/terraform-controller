package module

import (
	"context"
	"github.com/pkg/errors"
	"github.com/rancher/terraform-controller/pkg/apis/terraformcontroller.cattle.io/v1"
	"github.com/rancher/terraform-controller/pkg/digest"
	corev1 "github.com/rancher/terraform-controller/pkg/generated/controllers/core/v1"
	tfv1 "github.com/rancher/terraform-controller/pkg/generated/controllers/terraformcontroller.cattle.io/v1"
	"github.com/rancher/terraform-controller/pkg/git"
	"github.com/rancher/terraform-controller/pkg/interval"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"time"
)

func NewHandler(ctx context.Context, modules tfv1.ModuleController, secrets corev1.SecretController) *handler {
	return &handler{
		ctx:     ctx,
		modules: modules,
		secrets: secrets,
	}
}

type handler struct {
	ctx     context.Context
	modules tfv1.ModuleController
	secrets corev1.SecretController
}

func (h *handler) OnChange(key string, module *v1.Module) (*v1.Module, error) {
	if module == nil {
		return nil, nil
	}

	if isPolling(module.Spec) && needsUpdate(module) {
		v1.ModuleConditionGitUpdated.False(module)
		return tfv1.UpdateModuleOnChange(func(module runtime.Object) (runtime.Object, error) {
			v1.ModuleConditionGitUpdated.True(module)
			return h.modules.Update(module.(*v1.Module))
		}, h.updateCommit)(key, module)
	}
	//
	hash := computeHash(module)
	if module.Status.ContentHash != hash {
		return h.updateHash(module, hash)
	}

	return h.modules.Update(module)
	//return module, nil
}

func (h *handler) OnRemove(key string, module *v1.Module) (*v1.Module, error) {
	//nothing to do here
	return module, nil
}

func (h *handler) updateHash(module *v1.Module, hash string) (*v1.Module, error) {
	module = module.DeepCopy()
	module.Status.Content = module.Spec.ModuleContent
	module.Status.ContentHash = hash
	if isPolling(module.Spec) && module.Status.GitChecked != nil {
		module.Status.Content.Git.Commit = module.Status.GitChecked.Commit
	}
	return h.modules.Update(module)
}

func (h *handler) updateCommit(key string, module *v1.Module) (*v1.Module, error) {
	branch := module.Spec.Git.Branch
	if branch == "" {
		branch = "master"
	}

	auth, err := h.getAuth(module.Namespace, module.Spec)
	if err != nil {
		return nil, err
	}

	commit, err := git.BranchCommit(h.ctx, module.Spec.Git.URL, branch, &auth)
	if err != nil {
		return nil, err
	}

	gitChecked := module.Spec.Git
	gitChecked.Commit = commit
	module.Status.GitChecked = &gitChecked
	module.Status.CheckTime = metav1.NewTime(time.Now())

	return module, nil
}

func (h *handler) getAuth(ns string, spec v1.ModuleSpec) (git.Auth, error) {
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
		!v1.ModuleConditionGitUpdated.IsTrue(m) ||
		m.Status.GitChecked == nil ||
		m.Status.GitChecked.URL != m.Spec.Git.URL ||
		m.Status.GitChecked.Branch != m.Spec.Git.Branch
}

func isPolling(spec v1.ModuleSpec) bool {
	return len(spec.Content) == 0 &&
		spec.Git.URL != "" &&
		spec.Git.Commit == "" &&
		spec.Git.Tag == ""
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
