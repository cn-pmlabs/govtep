.ONESHELL:
SHELL = /bin/bash

GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
DIST=dist
BINARY_NAME=controller

export GO111MODULE=on
export GOPROXY=https://goproxy.io

all:govtep

.phony: all clean

odbgen:
	@echo "generate odbapi by schema"
	pushd ./cmd/odbgen
	$(GOCMD) run odbgen.go  -f schema/configdb.ovsschema
	$(GOCMD) run odbgen.go  -f schema/controller_vtep.ovsschema
	$(GOCMD) run odbgen.go  -f schema/ovn-nb.ovsschema 
	$(GOCMD) run odbgen.go  -f schema/ovn-sb.ovsschema
	popd

clean:
	$(GOCLEAN)
	rm -f $(DIST)/$(BINARY_NAME)

controller:
	$(GOBUILD) -v -gcflags "-N -l" -o $(DIST)/$(BINARY_NAME) cmd/controller/controller.go

govtep: odbgen controller
