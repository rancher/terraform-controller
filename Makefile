TARGETS := $(shell ls scripts)

.dapper:
	@echo Downloading dapper
	@curl -sL https://releases.rancher.com/dapper/latest/dapper-`uname -s`-`uname -m` > .dapper.tmp
	@@chmod +x .dapper.tmp
	@./.dapper.tmp -v
	@mv .dapper.tmp .dapper

$(TARGETS): .dapper
	./.dapper $@

deps: .dapper
	./.dapper -m bind env GO111MODULE=on go mod tidy
	./.dapper -m bind env GO111MODULE=on go mod vendor
	./.dapper -m bind chown -R $$(id -u) vendor dist bin go.mod go.sum .cache

.DEFAULT_GOAL := ci

.PHONY: $(TARGETS)
