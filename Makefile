NAMESPACE=default
IMAGENAME=quay.io/example/pipeline-operator
VERSION=0.0.1

# kernel-style V=1 build verbosity
ifeq ("$(origin V)", "command line")
       BUILD_VERBOSE = $(V)
endif

ifeq ($(BUILD_VERBOSE),1)
       Q =
else
       Q = @
endif

PKGS = $(shell go list ./... | grep -v /vendor/)

all: dep k8s build 
	

format:
	$(Q)go fmt $(PKGS)

dep:
	$(Q)dep ensure -v

dep-update:
	$(Q)dep ensure -update -v

clean:
	$(Q)rm -rf build

build: 
	operator-sdk build $(IMAGENAME)

test: dep k8s
	operator-sdk up local --namespace=$(NAMESPACE)

k8s:
	sed -i 's|REPLACE_IMAGE|$(IMAGENAME):$(VERSION)|g' deploy/operator.yaml
	sed -i "s|REPLACE_NAMESPACE|$(NAMESPACE)|g" deploy/role_binding.yaml
	- kubectl create -f deploy/service_account.yaml
	- kubectl create -f deploy/role.yaml
	- kubectl create -f deploy/role_binding.yaml
	- kubectl create -f deploy/operator.yaml

.PHONY: all test format dep clean k8s
