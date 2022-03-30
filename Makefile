.DEFAULT_GOAL := all

GO_VERSION_MANIFEST       := https://raw.githubusercontent.com/actions/go-versions/main/versions-manifest.json
#TODO bump the percentage to 60 (minimum) as soon as possible
REQUIRED_COVERAGE_PERCENT := 0
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

generate:
	# TODO add mocks and other generated logic here.

checkGenerated: generate
	@if git status --porcelain | grep -e [.]go -e [.]json -e [.]yaml; then \
		echo "Please commit generated files, using 'make generate'."; \
		git --no-pager diff; \
		exit 1; \
	fi; \
	true;

build:
	go build -tags=go_json -race -v ./...

buildAllSupportedPlatforms: clean
	echo "Linux"
	for arch in arm64 amd64 s390x ppc64le riscv64; do \
		CGO_ENABLED=0 GOOS=linux GOARCH=$$arch go build -tags=go_json -a -v ./...; \
	done;
	echo "Darwin"
	for arch in arm64 amd64; do \
		CGO_ENABLED=0 GOOS=darwin GOARCH=$$arch go build -tags=go_json -a -v ./...; \
	done;
	echo "Windows"
	for arch in arm64 amd64; do \
		CGO_ENABLED=0 GOOS=windows GOARCH=$$arch go build -tags=go_json -a -v ./...; \
	done;

test:
	@go version
	# TODO make -race work
	#go test -tags=go_json -v -race -cover -coverprofile=$(COVERAGE_FILE) -covermode atomic ./...
	go test -tags=go_json -v -cover -coverprofile=$(COVERAGE_FILE) -covermode atomic ./...
	@grep -v "_generated.go" $(COVERAGE_FILE) > tmp$(COVERAGE_FILE)
	@mv -f tmp$(COVERAGE_FILE) $(COVERAGE_FILE)

# TODO should be improved to a per file check and maybe against a previous value
#(maybe we should use something like SonarQube for this?)
coverage: $(COVERAGE_FILE)
	@t=`go tool cover -func=$(COVERAGE_FILE) | grep total | grep -Eo '[0-9]+\.[0-9]+'`;\
	echo "Total coverage: $${t}%"; \
	if [ "$${t%.*}" -lt $(REQUIRED_COVERAGE_PERCENT) ]; then \
		echo "ERROR: It has to be at least $(REQUIRED_COVERAGE_PERCENT)%"; \
		exit 1; \
	fi;

benchmark:
	# TODO make -race work
	go test -tags=go_json -run=^$ -v -bench=. -benchmem -benchtime 10s ./...

clean:
	@go clean
	@rm -f tmp$(COVERAGE_FILE) $(COVERAGE_FILE) 2>/dev/null || true

lint:
	golangci-lint run

getAddLicense:
	GO111MODULE=off go get -v -u github.com/google/addlicense

addLicense: getAddLicense
	`go env GOPATH`/bin/addlicense -f LICENSE *

checkLicense: getAddLicense
	`go env GOPATH`/bin/addlicense -f LICENSE -check *

all: checkLicense checkModVersion checkIfAllDependenciesAreUpToDate checkGenerated build buildAllSupportedPlatforms test coverage benchmark clean
local: addLicense checkLicense updateGoModVersion updateAllDependencies generate build test coverage benchmark lint clean