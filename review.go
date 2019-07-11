package review

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"

	"golang.org/x/build/gerrit"
)

const (
	ErrNoProblemsFound ErrorString = "no problems to convert into review comments"
	ErrSplitLine       ErrorString = "failed to split line to location and description"
	ErrSplitLocation   ErrorString = "failed split location to filename and position"
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

// LinesToReviewComments reads linter reports and converts those to Gerrit Review Comments.
// Key in returned map is file name and value all associated problems as comments.
func LinesToReviewComments(r io.Reader) (comments map[string][]gerrit.CommentInput, err error) {
	comments = map[string][]gerrit.CommentInput{}
	scanner := bufio.NewScanner(r)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()
		problem, err := parseLine(line)
		if err != nil {
			return nil, &ParseError{LineNumber: lineNumber, Err: err}
		}

		if _, ok := comments[problem.FileName]; !ok {
			comments[problem.FileName] = []gerrit.CommentInput{}
		}
		comments[problem.FileName] = append(comments[problem.FileName], gerrit.CommentInput{
			Line:    problem.Line,
			Message: problem.Description,
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if len(comments) == 0 {
		return nil, ErrNoProblemsFound
	}

	return comments, nil
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
