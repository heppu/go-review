package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/google/go-github/github"
	"github.com/heppu/go-review"
	"golang.org/x/oauth2"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"

	ver  = flag.Bool("version", false, "print versions details and exit")
	dry  = flag.Bool("dry", false, "parse env vars and input but do not publish review")
	show = flag.Bool("show", false, "print lines while parsing")
)

func main() {
	flag.Parse()

	if *ver {
		fmt.Fprintf(os.Stderr, "%v, commit %v, built at %v\n", version, commit, date)
		os.Exit(0)
	}

	owner := parseEnv("OWNER", true)
	repo := parseEnv("REPOSITORY", true)
	commit := parseEnv("COMMIT", true)
	prNumber := parseEnvInt("PULL_REQUEST", true)
	token := parseEnvInt("ACCESS_TOKEN", true)

	var input io.Reader = os.Stdin
	if *show {
		input = io.TeeReader(os.Stdin, os.Stdout)
	}

	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	c := github.NewClient(oauth2.NewClient(context.Background(), ts))

	diff, _, err := c.PullRequests.GetRaw(context.Background(), owner, repo, prNumber, github.RawOptions{github.Diff})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	comments, err := review.LinesToGithubComments(input, diff)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if len(comments) == 0 {
		return
	}

	fmt.Println(comments)
	if *dry {
		return
	}

	_, _, err = c.PullRequests.CreateReview(context.Background(), owner, repo, prNumber, &github.PullRequestReviewRequest{
		CommitID: github.String(commit),
		Body:     github.String(fmt.Sprintf("go-review reported %d problems", len(comments))),
		Event:    github.String("COMMENT"),
		Comments: comments,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
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

func parseEnvInt(name string, must bool) int {
	val, err := strconv.Atoi(parseEnv(name, must))
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s must be integer: %s\n", name, err)
		os.Exit(1)
	}
	return val
}
