package main

import (
	"github.com/rancher/wrangler/pkg/cleanup"
	"k8s.io/klog"
)

func main() {
	if err := cleanup.Cleanup("./pkg/apis"); err != nil {
		klog.Fatal(err)
	}
}
