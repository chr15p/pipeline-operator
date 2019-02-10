NAMESPACE=default
IMAGENAME=quay.io/example/pipeline-operator


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

all:
	

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

test: 
	operator-sdk up local --namespace=$(NAMESPACE)

.PHONY: all test format dep clean
