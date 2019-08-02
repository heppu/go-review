package review_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/go-github/github"
	"github.com/heppu/go-review"
	"github.com/stretchr/testify/require"
	"golang.org/x/build/gerrit"
)

func TestLinesToGerritCommentsSuccess(t *testing.T) {
	tests := []struct {
		name     string
		input    io.Reader
		comments map[string][]gerrit.CommentInput
	}{{
		name:  "SingleLine",
		input: strings.NewReader("file.go:1:2: some problem"),
		comments: map[string][]gerrit.CommentInput{
			"file.go": {{Line: 1, Message: "some problem"}},
		},
	}, {
		name: "MultiLineSingleFile",
		input: strings.NewReader(`file.go:1:2: some problem
file.go:2:2: other problem`),
		comments: map[string][]gerrit.CommentInput{
			"file.go": {
				{Line: 1, Message: "some problem"},
				{Line: 2, Message: "other problem"},
			},
		},
	}, {
		name: "MultiLineMultiFile",
		input: strings.NewReader(`file.go:1:2: some problem
file.go:2:2: other problem
file_2.go:3:5: problem`),
		comments: map[string][]gerrit.CommentInput{
			"file.go": {
				{Line: 1, Message: "some problem"},
				{Line: 2, Message: "other problem"},
			},
			"file_2.go": {{Line: 3, Message: "problem"}},
		},
	}, {
		name:  "MultiColons",
		input: strings.NewReader("file.go:1:2:3:4 problem"),
		comments: map[string][]gerrit.CommentInput{
			"file.go": {{Line: 1, Message: "problem"}},
		},
	}, {
		name:  "NoColumn",
		input: strings.NewReader("file.go:1 problem"),
		comments: map[string][]gerrit.CommentInput{
			"file.go": {{Line: 1, Message: "problem"}},
		},
	}, {
		name:  "DotGoInName",
		input: strings.NewReader("some.go/file.go:1 problem"),
		comments: map[string][]gerrit.CommentInput{
			"some.go/file.go": {{Line: 1, Message: "problem"}},
		},
	}, {
		name: "CommentLine",
		input: strings.NewReader(`# some/pkg
file.go:1:2: some problem`),
		comments: map[string][]gerrit.CommentInput{
			"file.go": {{Line: 1, Message: "some problem"}},
		},
	}, {
		name:     "EmptyInput",
		input:    strings.NewReader(``),
		comments: map[string][]gerrit.CommentInput{},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			comments, err := review.LinesToGerritComments(tt.input)
			require.NoError(t, err)
			require.Equal(t, tt.comments, comments)
		})
	}
}

type errorReader struct {
	err error
}

func (e errorReader) Read(_ []byte) (int, error) { return 0, e.err }

func TestLinesToGerritCommentsError(t *testing.T) {
	tests := []struct {
		name  string
		input io.Reader
		err   error
	}{{
		name:  "ReaderFailure",
		input: errorReader{err: errors.New("error")},
		err:   errors.New("error"),
	}, {
		name:  "NoDescription",
		input: strings.NewReader("file.go:1:1:"),
		err:   &review.ParseError{LineNumber: 1, Err: review.ErrSplitLine},
	}, {
		name:  "NoFileName",
		input: strings.NewReader("1:1: problem"),
		err:   &review.ParseError{LineNumber: 1, Err: review.ErrSplitLocation},
	}, {
		name:  "NoFileNameOrLine",
		input: strings.NewReader(" problem"),
		err:   &review.ParseError{LineNumber: 1, Err: review.ErrSplitLocation},
	}, {
		name:  "NonNumericLine",
		input: strings.NewReader("file.go:x:1: problem"),
		err:   &review.ParseError{LineNumber: 1, Err: errors.New("expected line number but got: x")},
	}, {
		name:  "NonNumericColumn",
		input: strings.NewReader("file.go:1:x: problem"),
		err:   &review.ParseError{LineNumber: 1, Err: errors.New("expected column number but got: x")},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := review.LinesToGerritComments(tt.input)
			require.EqualError(t, err, tt.err.Error())
		})
	}
}

func TestLinesToGithubCommentsSuccess(t *testing.T) {
	tests := []struct {
		name     string
		input    io.Reader
		diff     string
		comments []*github.DraftReviewComment
	}{{
		name: "MultiLineSingleFile",
		input: strings.NewReader(`cmd/go-gh-review/main.go:16:2: "version" is a global variable (gochecknoglobals)
cmd/go-gh-review/main.go:17:2: "commit" is a global variable (gochecknoglobals)
cmd/go-gh-review/main.go:18:2: "date" is a global variable (gochecknoglobals)
cmd/go-gh-review/main.go:20:2: "ver" is a global variable (gochecknoglobals)
cmd/go-gh-review/main.go:21:2: "dry" is a global variable (gochecknoglobals)
cmd/go-gh-review/main.go:22:2: "show" is a global variable (gochecknoglobals)
`),
		comments: []*github.DraftReviewComment{},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			comments, err := review.LinesToGerritComments(tt.input)
			require.NoError(t, err)
			require.Equal(t, tt.comments, comments)
		})
	}
}

func ExampleLinesToGerritComments() {
	// Mock Gerrit server
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := ioutil.ReadAll(r.Body)
		fmt.Printf("%s", data)
		fmt.Fprint(w, ")]}\n{}")
	}))
	defer s.Close()

	input := strings.NewReader(`file_1.go:1:2: some problem`)
	comments, err := review.LinesToGerritComments(input)
	if err != nil {
		log.Fatal(err)
	}

	r := gerrit.ReviewInput{
		Message:  "go-review",
		Comments: comments,
	}

	c := gerrit.NewClient(s.URL, gerrit.NoAuth)
	if err := c.SetReview(context.Background(), "some-change-id", "some-revision", r); err != nil {
		log.Fatal(err)
	}

	// {
	//   "message": "go-review",
	//   "comments": {
	//     "file_1.go": [
	//       {
	//         "line": 1,
	//         "message": "some problem"
	//       }
	//     ]
	//   }
	// }
}
