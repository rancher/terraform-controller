package module

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/rancher/kerraform/pkg/digest"
	"github.com/rancher/kerraform/pkg/git"
	"github.com/rancher/kerraform/pkg/interval"
	corev1client "github.com/rancher/kerraform/types/apis/core/v1"
	"github.com/rancher/kerraform/types/apis/kerraform.cattle.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func Register(ctx context.Context, ns string, client v1.Interface, k8sClient corev1client.Interface) error {
	fl := &handler{
		ctx:           ctx,
		modules:       client.Modules(""),
		secretsLister: k8sClient.Secrets("").Controller().Lister(),
	}

	client.Modules(ns).AddHandler(ctx, "module controller", fl.Handler)
	return nil
}

type handler struct {
	ctx           context.Context
	modules       v1.ModuleInterface
	secretsLister corev1client.SecretLister
}

func (h *handler) Handler(key string, obj *v1.Module) (robj runtime.Object, rerr error) {
	if obj == nil {
		return nil, nil
	}

	if isPolling(obj.Spec) && needsUpdate(obj) {
		return v1.ModuleConditionGitUpdated.Track(obj, h.modules, func() (runtime.Object, error) {
			return h.updateCommit(obj)
		})
	}

	hash := computeHash(obj)
	if obj.Status.ContentHash != hash {
		return h.updateHash(obj, hash)
	}

	return obj, nil
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

func (h *handler) updateCommit(module *v1.Module) (*v1.Module, error) {
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

	// copy
	gitChecked := module.Spec.Git
	gitChecked.Commit = commit

	module = module.DeepCopy()
	module.Status.GitChecked = &gitChecked
	module.Status.CheckTime = metav1.NewTime(time.Now())
	return h.modules.Update(module)
}

func (h *handler) getAuth(ns string, spec v1.ModuleSpec) (git.Auth, error) {
	auth := git.Auth{}
	name := spec.Git.SecretName

	if name == "" {
		return auth, nil
	}

	secret, err := h.secretsLister.Get(ns, name)
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
