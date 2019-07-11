package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/heppu/go-review"
	"golang.org/x/build/gerrit"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	v := flag.Bool("version", false, "print versions details and exit")
	flag.Parse()
	if *v {
		fmt.Printf("%v, commit %v, built at %v", version, commit, date)
		os.Exit(0)
	}

	reviewURL := parseEnv("GERRIT_REVIEW_URL", true)
	changeID := parseEnv("GERRIT_CHANGE_ID", true)
	revision := parseEnv("GERRIT_PATCHSET_REVISION", true)
	username := parseEnv("GERRIT_USERNAME", false)
	password := parseEnv("GERRIT_PASSWORD", false)

	var auth gerrit.Auth = gerrit.NoAuth
	if username != "" && password != "" {
		auth = gerrit.BasicAuth(username, password)
	}

	comments, err := review.LinesToReviewComments(os.Stdin)
	if err != nil {
		fmt.Println(err)
		if err == review.ErrNoProblemsFound {
			os.Exit(0)
		}
		os.Exit(1)
	}
	r := gerrit.ReviewInput{
		Message:  "go-review",
		Comments: comments,
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
		fmt.Printf("%s must be set", name)
		os.Exit(1)
	}
	return val
}
