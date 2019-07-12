[![GoDoc](https://godoc.org/github.com/heppu/go-review?status.svg)](https://godoc.org/github.com/heppu/go-review)
[![Build Status](https://travis-ci.com/heppu/go-review.svg?branch=master)](https://travis-ci.com/heppu/go-review)
[![Coverage Status](https://coveralls.io/repos/github/heppu/go-review/badge.svg?branch=master)](https://coveralls.io/github/heppu/go-review?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/heppu/go-review)](https://goreportcard.com/report/github.com/heppu/go-review)

# Go Review
Publish reports from different Go linters as gerrit review comments.

## Install

Specific release can be installed by downloading from [releases page](https://github.com/heppu/go-review/releases).

Latest version from master can be installed by running: `go get github.com/heppu/go-review/...`

## Usage

go-review uses following environment variables to access gerrit review server:

- `GERRIT_REVIEW_URL`: required
- `GERRIT_CHANGE_ID`: required
- `GERRIT_PATCHSET_REVISION`: required
- `GERRIT_USERNAME`: optional
- `GERRIT_PASSWORD`: optional

Behavior can be controlled with following flags:

 - `-version`: print versions details and exit
 - `-dry`: parse env vars and input but do not publish review
 - `-show`: print lines while parsing

## Examples
Every linter that is able to produce similar output as go vet can be used as an input.

### go vet

```sh
go vet ./... 2>&1 | go-review
```

### golangci-lint

```sh
 golangci-lint run --out-format=line-number --print-issued-lines=false | go-review
```

### staticcheck

```sh
staticcheck ./... | go-review
```
