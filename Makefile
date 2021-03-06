BIN_FILE = target/bin/go-review
COVERAGE_FILE = target/test/coverage.txt

.PHONY: build
build:
	@mkdir -p $(dir ${BIN_FILE})
	go build -o ${BIN_FILE} cmd/go-review/main.go

.PHONY: test
test:
	@mkdir -p $(dir ${COVERAGE_FILE})
	go test -v -cover -race -coverprofile=${COVERAGE_FILE}

.PHONY: lint
lint:
	golangci-lint run --tests=false --enable-all --disable=lll,gochecknoglobals,wsl,gomnd ./...

.PHONY: goveralls
goveralls: test
	goveralls -v -service=travis-ci -coverprofile=${COVERAGE_FILE}
