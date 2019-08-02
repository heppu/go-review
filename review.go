package review

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/google/go-github/github"
	"github.com/waigani/diffparser"
	"golang.org/x/build/gerrit"
)

const (
	ErrSplitLine     ErrorString = "failed to split line to location and description"
	ErrSplitLocation ErrorString = "failed split location to filename and position"
)

type ErrorString string

func (e ErrorString) Error() string { return string(e) }

type Problem struct {
	FileName    string
	Description string
	Position
}

type Position struct {
	Line   int
	Column int
}

type ParseError struct {
	LineNumber int
	Err        error
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("could not parse line: %d, %s", e.LineNumber, e.Err)
}

// LinesToProblems reads linter reports and converts those to slice of Problem structs.
func LinesToProblems(r io.Reader) (problems []Problem, err error) {
	problems = []Problem{}
	err = iterateLines(r, func(p Problem) { problems = append(problems, p) })
	if err != nil {
		return nil, err
	}

	return problems, nil
}

// LinesToGerritComments reads linter reports and converts those to Gerrit Review Comments.
// Key in returned map is file name and value all associated problems as comments.
func LinesToGerritComments(r io.Reader) (comments map[string][]gerrit.CommentInput, err error) {
	comments = map[string][]gerrit.CommentInput{}
	err = iterateLines(r, func(p Problem) {
		if _, ok := comments[p.FileName]; !ok {
			comments[p.FileName] = []gerrit.CommentInput{}
		}
		comments[p.FileName] = append(comments[p.FileName], gerrit.CommentInput{
			Line:    p.Line,
			Message: p.Description,
		})
	})
	if err != nil {
		return nil, err
	}

	return comments, nil
}

// LinesToGithubComments reads linter reports and converts those to slice of Github Review Comments.
// PR's diff is needed to insert comments in correct lines since that's how github API wants the position.
// Problems that are not part of diff will be ignored.
func LinesToGithubComments(r io.Reader, diff string) (comments []*github.DraftReviewComment, err error) {
	lines, err := diffToLineMap(diff)
	if err != nil {
		return nil, err
	}

	comments = []*github.DraftReviewComment{}
	err = iterateLines(r, func(p Problem) {
		if f, ok := lines[p.FileName]; ok {
			if position, ok := f[p.Line]; ok {
				comments = append(comments, &github.DraftReviewComment{
					Path:     github.String(p.FileName),
					Body:     github.String(p.Description),
					Position: github.Int(position),
				})
			}
		}
	})
	if err != nil {
		return nil, err
	}

	return comments, nil
}

func diffToLineMap(diff string) (map[string]map[int]int, error) {
	d, err := diffparser.Parse(diff)
	if err != nil {
		return nil, err
	}

	lines := map[string]map[int]int{}
	for _, f := range d.Files {
		lines[f.NewName] = map[int]int{}
		for _, h := range f.Hunks {
			for _, l := range h.NewRange.Lines {
				lines[f.NewName][l.Number] = l.Position - 1
			}
		}
	}
	return lines, nil
}

func iterateLines(r io.Reader, callback func(p Problem)) (err error) {
	scanner := bufio.NewScanner(r)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()

		if strings.HasPrefix(line, "#") {
			continue
		}

		problem, err := parseLine(line)
		if err != nil {
			return &ParseError{LineNumber: lineNumber, Err: err}
		}
		callback(problem)
	}

	return scanner.Err()
}

func parseLine(line string) (Problem, error) {
	report := Problem{}
	parts := strings.SplitN(line, " ", 2)
	if len(parts) != 2 {
		return report, ErrSplitLine
	}

	locParts := strings.SplitAfter(parts[0], ".go")
	if len(locParts) < 2 {
		return Problem{}, ErrSplitLocation
	}
	report.FileName = strings.Join(locParts[0:len(locParts)-1], "")

	pos, err := parsePosition(locParts[len(locParts)-1])
	if err != nil {
		return report, err
	}

	report.Position = pos
	report.Description = parts[1]
	return report, nil
}

func parsePosition(text string) (pos Position, err error) {
	parts := strings.FieldsFunc(text, func(c rune) bool { return c == ':' })
	switch {
	case len(parts) > 1:
		i, err := strconv.Atoi(parts[1])
		if err != nil {
			return pos, fmt.Errorf("expected column number but got: %s", parts[1])
		}
		pos.Column = i
		fallthrough
	case len(parts) > 0:
		i, err := strconv.Atoi(parts[0])
		if err != nil {
			return pos, fmt.Errorf("expected line number but got: %s", parts[0])
		}
		pos.Line = i
	}
	return pos, nil
}
