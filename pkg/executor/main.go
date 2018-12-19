package main

import (
	"log"
	"os"

	"github.com/ibuildthecloud/terraform-operator/pkg/executor/runner"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/rest"
)

func main() {
	if os.Getenv("EXECUTOR_DEBUG") == "true" {
		logrus.SetLevel(logrus.DebugLevel)
	}

	err := run()

	if err != nil {
		log.Fatal(err)
	}
}

func run() error {
	config, err := rest.InClusterConfig()
	if err != nil {
		return err
	}

	runner, err := runner.NewRunner(config)
	if err != nil {
		return err
	}

	err = runner.Populate()
	if err != nil {
		return err
	}

	return nil
}
