.DEFAULT_GOAL := all

GO_VERSION_MANIFEST       := https://raw.githubusercontent.com/actions/go-versions/main/versions-manifest.json
REQUIRED_COVERAGE_PERCENT := 70
COVERAGE_FILE             := cover.out
REPOSITORY                := $(shell basename `pwd`)

CGO_ENABLED := 1
GOOS         ?=
GOARCH       ?=

export CGO_ENABLED GOOS GOARCH

define getLatestGoPatchVersion
	$(shell curl -s $(GO_VERSION_MANIFEST) | jq -r '.[0].version')
endef

define getLatestGoMinorVersion
	$(shell echo $(call getLatestGoPatchVersion) | cut -f1,2 -d'.')
endef

latestGoVersion:
	@echo $(call getLatestGoPatchVersion)

latestGoMinorVersion:
	@echo $(call getLatestGoMinorVersion)

updateGoModVersion:
	go mod edit -go $(call getLatestGoMinorVersion)

checkModVersion: updateGoModVersion
	@if git status --porcelain | grep -q go.mod; then \
		echo "Outdated go version in go.mod. Please update it using 'make updateGoModVersion' and make sure everything works correctly and tests pass then commit the changes."; \
		exit 1; \
	fi; \
	true;

updateAllDependencies:
	go get -t -u ./...
	go mod tidy

checkIfAllDependenciesAreUpToDate: updateAllDependencies
	@if git status --porcelain | grep -q go.sum; then \
		echo "Some dependencies are outdated. Please update all dependencies using 'make updateAllDependencies' and make sure everything works correctly and tests pass then commit the changes."; \
		exit 1; \
	fi; \
	true;

generate-mocks:
#	go install github.com/golang/mock/mockgen@latest
#	mockgen -source=CHANGE_ME.go -destination=CHANGE_ME.go -package=CHANGE_ME

generate:
	$(MAKE) generate-mocks
	$(MAKE) addLicense

checkGenerated: generate
	@if git status --porcelain | grep -e [.]go -e [.]json -e [.]yaml; then \
		echo "Please commit generated files, using 'make generate'."; \
		git --no-pager diff; \
		exit 1; \
	fi; \
	true;

build-all@ci/cd:
	go build -tags=go_json -v -race ./...

build: build-all@ci/cd

test:
	set -xe; \
	mf="$$(pwd)/Makefile"; \
	find . -mindepth 1 -maxdepth 4 -type d -print | grep -v '\./\.' | grep -v '/\.' | sed 's/\.\///g' | while read service; do \
		cd $${service} ; \
		if [[ $$(find . -mindepth 1 -maxdepth 1 -type f -print | grep -E '_test.go' | wc -l | sed "s/ //g") -gt 0 ]]; then \
			make -f $$mf test@ci/cd; \
		fi ; \
		for ((i=0;i<$$(echo "$${service}" | grep -o "/" | wc -l | sed "s/ //g");i++)); do \
          	cd .. ; \
        done; \
        cd .. ; \
	done;

# TODO should be improved to a per file check and maybe against a previous value
#(maybe we should use something like SonarQube for this?)
coverage: $(COVERAGE_FILE)
	@t=`go tool cover -func=$(COVERAGE_FILE) | grep total | grep -Eo '[0-9]+\.[0-9]+'`;\
	echo "Total coverage: $${t}%"; \
	if [ "$${t%.*}" -lt $(REQUIRED_COVERAGE_PERCENT) ]; then \
		echo "ERROR: It has to be at least $(REQUIRED_COVERAGE_PERCENT)%"; \
		exit 1; \
	fi;

test@ci/cd:
	# TODO make -race work
	go test -tags=go_json -v -cover -coverprofile=$(COVERAGE_FILE) -covermode atomic

benchmark@ci/cd:
	# TODO make -race work
	go test -tags=go_json -run=^$ -v -bench=. -benchmem -benchtime 10s

benchmark:
	set -xe; \
	mf="$$(pwd)/Makefile"; \
	find . -mindepth 1 -maxdepth 4 -type d -print | grep -v '\./\.' | grep -v '/\.' | sed 's/\.\///g' | while read service; do \
		cd $${service} ; \
		if [[ $$(find . -mindepth 1 -maxdepth 1 -type f -print | grep -E '_test.go' | wc -l | sed "s/ //g") -gt 0 ]]; then \
			make -f $$mf benchmark@ci/cd; \
		fi ; \
		for ((i=0;i<$$(echo "$${service}" | grep -o "/" | wc -l | sed "s/ //g");i++)); do \
          	cd .. ; \
        done; \
        cd .. ; \
	done;

print-all-packages-with-tests:
	set -xe; \
	find . -mindepth 1 -maxdepth 4 -type d -print | grep -v '\./\.' | grep -v '/\.' | sed 's/\.\///g' | while read service; do \
		cd $${service} ; \
		if [[ $$(find . -mindepth 1 -maxdepth 1 -type f -print | grep -E '_test.go' | wc -l | sed "s/ //g") -gt 0 ]]; then \
			echo "$${service}"; \
		fi ; \
		for ((i=0;i<$$(echo "$${service}" | grep -o "/" | wc -l | sed "s/ //g");i++)); do \
          	cd .. ; \
        done; \
        cd .. ; \
	done;

clean:
	@go clean
	@rm -f tmp$(COVERAGE_FILE) $(COVERAGE_FILE) 2>/dev/null || true
	@find . -name ".tmp-*" -exec rm -Rf {} \; || true;
	@find . -mindepth 1 -maxdepth 2 -type f -name $(COVERAGE_FILE) -exec rm -Rf {} \; || true;
	@find . -mindepth 1 -maxdepth 2 -type f -name tmp$(COVERAGE_FILE) -exec rm -Rf {} \; || true;

lint:
	golangci-lint run

getAddLicense:
	go install -v github.com/google/addlicense@latest

addLicense: getAddLicense
	`go env GOPATH`/bin/addlicense -f LICENSE.header * .github/*

checkLicense: getAddLicense
	`go env GOPATH`/bin/addlicense -f LICENSE.header -check * .github/*

fix-field-alignment:
	go install golang.org/x/tools/go/analysis/passes/fieldalignment/cmd/fieldalignment@latest
	fieldalignment -fix ./...

format-imports:
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/daixiang0/gci@latest
	gci write -s standard -s default -s "prefix(github.com/ice-blockchain)" ./..
	goimports -w -local github.com/ice-blockchain ./..

all: checkLicense checkModVersion checkIfAllDependenciesAreUpToDate checkGenerated build test coverage benchmark clean
local: addLicense updateGoModVersion updateAllDependencies generate build test coverage benchmark lint clean
