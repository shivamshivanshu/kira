package core

import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/id"
	"github.com/shivamshivanshu/kira/internal/rgx"
)

type RowKind int

const (
	RowMatch RowKind = iota
	RowContext
	RowSeparator
)

type FindRow struct {
	Kind   RowKind
	ID     string
	Number string
	Line   int
	Text   string
}

func NewFindResult(rows []FindRow) datamodel.FindResult {
	matches := make([]datamodel.Match, 0, len(rows))
	for _, r := range rows {
		if r.Kind == RowMatch {
			matches = append(matches, datamodel.Match{ID: r.ID, Number: r.Number, Line: r.Line, Text: r.Text})
		}
	}
	// Both backends append a file's matches in ascending line order and
	// sortByKey is stable, so hits within one item stay line-ordered with no
	// explicit tiebreak — a backend that appended out of order would need one.
	sortByKey(matches, func(m datamodel.Match) id.SortKey { return id.NewSortKey(m.Number, m.ID) })
	return datamodel.FindResult{Matches: matches}
}

type FindArgs struct {
	Pattern    string
	Passthru   []string
	IgnoreCase bool
	Word       bool
}

var rgFlagsTakingValue = map[string]bool{
	"-A": true, "--after-context": true,
	"-B": true, "--before-context": true,
	"-C": true, "--context": true,
	"-m": true, "--max-count": true,
	"-e": true, "--regexp": true,
	"-f": true, "--file": true,
	"--replace": true,
}

func ParseFindArgs(args, dropExact []string) FindArgs {
	drop := make(map[string]bool, len(dropExact))
	for _, d := range dropExact {
		drop[d] = true
	}
	var fa FindArgs
	nextArgIsFlagValue := false
	for _, a := range args {
		if drop[a] {
			continue
		}
		fa.Passthru = append(fa.Passthru, a)
		if nextArgIsFlagValue {
			nextArgIsFlagValue = false
			continue
		}
		switch {
		case a == "-i" || a == "--ignore-case":
			fa.IgnoreCase = true
		case a == "-w" || a == "--word-regexp":
			fa.Word = true
		case rgFlagsTakingValue[a]:
			nextArgIsFlagValue = true
		case !strings.HasPrefix(a, "-") && fa.Pattern == "":
			fa.Pattern = a
		}
	}
	return fa
}

func (s *Store) Find(cfg *datamodel.Config, args FindArgs) ([]FindRow, error) {
	items, err := s.LoadAll()
	if err != nil {
		return nil, err
	}
	if rgx.Installed() {
		return s.findRipgrep(args, items)
	}
	return s.findFallback(args, items)
}

func (s *Store) findRipgrep(args FindArgs, items []*datamodel.Item) ([]FindRow, error) {
	byULID := make(map[string]*datamodel.Item, len(items))
	for _, it := range items {
		byULID[it.ID] = it
	}

	fs := s.fs()
	rgArgs := make([]string, 0, len(args.Passthru)+2)
	rgArgs = append(rgArgs, args.Passthru...)
	rgArgs = append(rgArgs, "--", fs.RelToRoot(fs.TicketsDir()))

	lines, matched, err := rgx.Search(fs.Root(), rgArgs)
	if err != nil {
		return nil, errx.User("find: %s", err)
	}
	if !matched {
		return nil, nil
	}

	var rows []FindRow
	for _, l := range lines {
		if l.Path == "" {
			rows = append(rows, FindRow{Kind: RowSeparator, Text: l.Text})
			continue
		}
		number, itemID := l.Path, ""
		if it := byULID[ulidFromPath(l.Path)]; it != nil {
			number, itemID = it.Number, it.ID
		}
		if l.IsMatch {
			rows = append(rows, FindRow{Kind: RowMatch, ID: itemID, Number: number, Line: l.LineNo, Text: l.Text})
		} else {
			rows = append(rows, FindRow{Kind: RowContext, Number: number, Line: l.LineNo, Text: l.Text})
		}
	}
	return rows, nil
}

func (s *Store) findFallback(args FindArgs, items []*datamodel.Item) ([]FindRow, error) {
	if args.Pattern == "" {
		return nil, errx.User("find: a search pattern is required")
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
	buf := make([]byte, 0, 64*1024)
	fs := s.fs()
	for _, it := range items {
		data, err := os.ReadFile(fs.ItemPath(it.ID))
		if err != nil {
			return nil, errx.User("reading %s: %v", it.Number, err)
		}
		lineNo := 0
		scanner := bufio.NewScanner(bytes.NewReader(data))
		scanner.Buffer(buf, 4*1024*1024)
		for scanner.Scan() {
			lineNo++
			line := scanner.Text()
			if re.MatchString(line) {
				rows = append(rows, FindRow{Kind: RowMatch, ID: it.ID, Number: it.Number, Line: lineNo, Text: line})
			}
		}
	}
	return rows, nil
}

func ulidFromPath(path string) string {
	return strings.TrimSuffix(filepath.Base(path), ".md")
}
