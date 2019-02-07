# This file is the development Makefile for the bully-algorithm project.
# All variables listed here are used as substitution in these Makefile targets.

SERVICE-NAME = bully-algorithm

define ENV-CONFIGURATION
ENV='dev'
endef

################################################################################


# Install all dependencies required.
#
# NOTE: Docker & Docker Compose should already be installed.
.PHONY: install
install:
		curl https://glide.sh/get | sh
		go get -u github.com/alecthomas/gometalinter
		gometalinter --install
		glide update
		glide install

# Build project binaries.
.PHONY: build
build:
		cd cmd/bully && go build -o bully
		cd cmd/data-viz && go build -o data-viz

# Runs linter against the service codebase.
#
# NOTE: This rule require gcc to be found in the `$PATH`.
.PHONY: lint
lint:
		@gometalinter --config=conf/gometalinter_conf.json ./... && \
		echo "linter pass ok !"

# Runs test suite.
.PHONY: test
test: lint
		go test -tags integration -race -cover -timeout=120s $$(glide novendor)
