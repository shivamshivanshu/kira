package core

import (
	"bufio"
	"bytes"
	"os"
	"regexp"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/id"
)

type FindRow struct {
	ID     string
	Number string
	Line   int
	Text   string
}

func NewFindResult(rows []FindRow) datamodel.FindResult {
	matches := make([]datamodel.Match, 0, len(rows))
	for _, r := range rows {
		matches = append(matches, datamodel.Match{ID: r.ID, Number: r.Number, Line: r.Line, Text: r.Text})
	}
	// findInItems appends a file's matches in ascending line order and
	// sortByKey is stable, so hits within one item stay line-ordered with no
	// explicit tiebreak — a search that appended out of order would need one.
	sortByKey(matches, func(m datamodel.Match) id.SortKey { return id.NewSortKey(m.Number, m.ID) })
	return datamodel.FindResult{Matches: matches}
}

type FindArgs struct {
	Pattern    string
	IgnoreCase bool
	Word       bool
}

const (
	scannerInitialBuffer = 64 * 1024
	scannerMaxLineSize   = 4 * 1024 * 1024
)

func (s *Store) Find(cfg *datamodel.Config, args FindArgs) ([]FindRow, error) {
	ld, err := s.read(cfg, loadOpts{})
	if err != nil {
		return nil, err
	}
	return s.findInItems(args, ld.items)
}

func (s *Store) findInItems(args FindArgs, items []*datamodel.Item) ([]FindRow, error) {
	if args.Pattern == "" {
		return nil, errx.User("find: a search pattern is required").WithHint("example: kira find \"race condition\"")
	}
	expr := args.Pattern
	if args.Word {
		expr = `\b(?:` + expr + `)\b`
	}
	if args.IgnoreCase {
		expr = `(?i)` + expr
	}
	re, err := regexp.Compile(expr)
	if err != nil {
		return nil, errx.User("find: invalid pattern: %v", err)
	}

	var rows []FindRow
	buf := make([]byte, 0, scannerInitialBuffer)
	fs := s.fs()
	for _, it := range items {
		data, err := os.ReadFile(fs.ItemPath(it.ID))
		if err != nil {
			return nil, errx.User("reading %s: %v", it.Number, err)
		}
		lineNo := 0
		scanner := bufio.NewScanner(bytes.NewReader(data))
		scanner.Buffer(buf, scannerMaxLineSize)
		for scanner.Scan() {
			lineNo++
			line := scanner.Text()
			if re.MatchString(line) {
				rows = append(rows, FindRow{ID: it.ID, Number: it.Number, Line: lineNo, Text: line})
			}
		}
		if err := scanner.Err(); err != nil {
			return rows, errx.User("reading %s: %v", it.Number, err)
		}
	}
	return rows, nil
}
