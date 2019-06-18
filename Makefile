.DEFAULT_GOAL := all

build-agents:
	@echo "Building supported agent binaries"
	@if [ ! -d "../service-net-agent" ]; then \
		echo "service-net-agent repository not found, please clone it first"; \
		exit 1; \
	fi
	@cd ../service-net-agent && $(MAKE) dep && $(MAKE) build-custom BUILDOS=windows BUILDARCH=amd64 && cd ../edge-controller
	@cd ../service-net-agent && $(MAKE) dep && $(MAKE) build-custom BUILDOS=linux BUILDARCH=amd64 && cd ../edge-controller
	@cd ../service-net-agent && $(MAKE) dep && $(MAKE) build-custom BUILDOS=darwin BUILDARCH=amd64 && cd ../edge-controller

vagrant: dep build-custom build-agents vagrant-up
vagrant-rebuild: dep build-custom build-agents vagrant-restart-service

include scripts/Makefile.common
include scripts/Makefile.vagrant
include scripts/Makefile.docker
include scripts/Makefile.k8s
include scripts/Makefile.azure
include scripts/Makefile.golang
