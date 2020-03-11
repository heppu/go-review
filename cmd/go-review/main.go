package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/heppu/go-review"
	"golang.org/x/build/gerrit"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"

	ver  = flag.Bool("version", false, "print versions details and exit")
	dry  = flag.Bool("dry-run", false, "parse env vars and input but do not publish review")
	show = flag.Bool("show", false, "print lines while parsing")
)

func main() {
	flag.Parse()

	if *ver {
		fmt.Fprintf(os.Stderr, "%v, commit %v, built at %v\n", version, commit, date)
		os.Exit(0)
	}

	reviewURL := parseEnv("GERRIT_REVIEW_URL", true)
	changeID := parseEnv("GERRIT_CHANGE_NUMBER", true)
	revision := parseEnv("GERRIT_PATCHSET_NUMBER", true)
	username := parseEnv("GERRIT_USERNAME", false)
	password := parseEnv("GERRIT_PASSWORD", false)

	var auth gerrit.Auth = gerrit.NoAuth
	if username != "" && password != "" {
		auth = gerrit.BasicAuth(username, password)
	}

	var input io.Reader = os.Stdin
	if *show {
		input = io.TeeReader(os.Stdin, os.Stdout)
	}

	comments, err := review.LinesToReviewComments(input)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		if err == review.ErrNoProblemsFound {
			return
		}
		os.Exit(1)
	}
	r := gerrit.ReviewInput{
		Message:  "go-review",
		Comments: comments,
	}

	if *dry {
		return
	}

	c := gerrit.NewClient(reviewURL, auth)
	if err := c.SetReview(context.Background(), changeID, revision, r); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func parseEnv(name string, must bool) string {
	val := os.Getenv(name)
	if must && val == "" {
		fmt.Fprintf(os.Stderr, "%s must be set\n", name)
		os.Exit(1)
	}
	return val
}
