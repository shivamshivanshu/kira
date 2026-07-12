package core

import (
	"bufio"
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/shivamshivanshu/kira/internal/config"
	"github.com/shivamshivanshu/kira/internal/id"
	"github.com/shivamshivanshu/kira/internal/item"
)

// RowKind distinguishes the kinds of line find emits. Only RowMatch rows carry
// an item identity and reach the --json contract; RowContext and RowSeparator
// are rg's -C context output, rendered for humans but never in JSON.
type RowKind int

const (
	RowMatch RowKind = iota
	RowContext
	RowSeparator
)

// FindRow is one line of find output from either backend. It is the single data
// shape both the rg and the pure-Go path produce, so there is one renderer and
// one JSON builder (docs/design/04-cli.md find). The fallback emits only
// RowMatch rows; the rg path also emits context/separator rows under -C.
type FindRow struct {
	Kind   RowKind
	ID     string
	Number string
	Line   int
	Text   string
}

// Match is one full-text hit in the find --json contract (docs/design/04-cli.md find).
type Match struct {
	ID     string `json:"id"`
	Number string `json:"number"`
	Line   int    `json:"line"`
	Text   string `json:"text"`
}

// FindResult is the find --json envelope.
type FindResult struct {
	Matches []Match `json:"matches"`
}

// NewFindResult projects the match rows into the frozen JSON envelope, sorted by
// the shared display-number key (docs/design/04-cli.md §7) so both backends emit
// identically ordered JSON. Context/separator rows are dropped.
func NewFindResult(rows []FindRow) FindResult {
	matches := make([]Match, 0, len(rows))
	for _, r := range rows {
		if r.Kind == RowMatch {
			matches = append(matches, Match{ID: r.ID, Number: r.Number, Line: r.Line, Text: r.Text})
		}
	}
	// Both backends append a file's matches in ascending line order and
	// sortByKey is stable, so hits within one item stay line-ordered with no
	// explicit tiebreak — a backend that appended out of order would need one.
	sortByKey(matches, func(m Match) id.SortKey { return id.NewSortKey(m.Number, m.ID) })
	return FindResult{Matches: matches}
}

// FindArgs is the parsed find invocation. Passthru forwards to rg verbatim
// (pattern included); IgnoreCase/Word are the subset the pure-Go fallback
// honors (docs/design/04-cli.md §5).
type FindArgs struct {
	Pattern    string
	Passthru   []string
	IgnoreCase bool
	Word       bool
}

// findValueFlags are the rg flags that consume a following argument. They are
// tracked only so the fallback's pattern detection skips a flag's value instead
// of mistaking it for the search pattern (e.g. `find -m 3 foo`).
var findValueFlags = map[string]bool{
	"-A": true, "--after-context": true,
	"-B": true, "--before-context": true,
	"-C": true, "--context": true,
	"-m": true, "--max-count": true,
	"-e": true, "--regexp": true,
	"-f": true, "--file": true,
	"--replace": true,
}

// ParseFindArgs splits raw find args into the rg passthrough plus the pattern
// and the -i/-w subset the fallback honors. dropExact names the kira global
// flags to strip (they are not rg flags); everything else forwards to rg
// verbatim. Detecting --json (a global) stays with the caller.
func ParseFindArgs(args, dropExact []string) FindArgs {
	drop := make(map[string]bool, len(dropExact))
	for _, d := range dropExact {
		drop[d] = true
	}
	var fa FindArgs
	skipVal := false
	for _, a := range args {
		if drop[a] {
			continue
		}
		fa.Passthru = append(fa.Passthru, a)
		if skipVal { // value of a preceding value-taking flag, not the pattern
			skipVal = false
			continue
		}
		switch {
		case a == "-i" || a == "--ignore-case":
			fa.IgnoreCase = true
		case a == "-w" || a == "--word-regexp":
			fa.Word = true
		case findValueFlags[a]:
			skipVal = true
		case !strings.HasPrefix(a, "-") && fa.Pattern == "":
			fa.Pattern = a
		}
	}
	return fa
}

// HaveRipgrep reports whether rg is on PATH, detected at call time per the
// external-tool policy (docs/design/01-architecture.md §7).
func HaveRipgrep() bool {
	_, err := exec.LookPath("rg")
	return err == nil
}

// rgLineRE matches one rg output line: `<path>(:|-)<lineno>(:|-)<text>`. A colon
// separator marks a real match; a dash marks a -C context line. The path never
// contains a colon (`.kira/tickets/<ulid>.md`), so the first colon-or-dash after
// `.md` is the field boundary.
var rgLineRE = regexp.MustCompile(`^(.*?\.md)([:-])(\d+)[:-](.*)$`)

// Find runs the free-text search over .kira/tickets/. With rg on PATH it shells
// out (full flag support); otherwise it falls back to a pure-Go regex scan over
// the same files — slower and without -C context, but the same result contract
// (docs/design/04-cli.md §5). Rows come back in backend-natural (scan) order;
// NewFindResult imposes the deterministic order for JSON.
func (s *Store) Find(cfg *config.Config, args FindArgs) ([]FindRow, error) {
	items, err := s.LoadAll()
	if err != nil {
		return nil, err
	}
	if HaveRipgrep() {
		return s.findRipgrep(args, items)
	}
	return s.findFallback(args, items)
}

// findRipgrep runs rg and parses its output into rows. rg exits 1 when there are
// no matches (not an error here) and 2 on a real failure (e.g. a bad regex),
// which maps to a user error (exit 1) per the exit-code policy.
func (s *Store) findRipgrep(args FindArgs, items []*item.Item) ([]FindRow, error) {
	byULID := make(map[string]*item.Item, len(items))
	for _, it := range items {
		byULID[it.ID] = it
	}

	rgArgs := append([]string{
		"--line-number", "--no-heading", "--with-filename", "--color=never",
	}, args.Passthru...)
	rgArgs = append(rgArgs, "--", filepath.Join(dirName, "tickets"))

	cmd := exec.Command("rg", rgArgs...)
	cmd.Dir = s.root
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			if ee.ExitCode() == 1 {
				return nil, nil // no matches
			}
			msg := strings.TrimSpace(string(ee.Stderr))
			if msg == "" {
				msg = "ripgrep failed"
			}
			return nil, userErr("find: %s", msg)
		}
		return nil, userErr("running ripgrep: %v", err)
	}

	var rows []FindRow
	scanner := bufio.NewScanner(bytes.NewReader(out))
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		m := rgLineRE.FindStringSubmatch(line)
		if m == nil {
			rows = append(rows, FindRow{Kind: RowSeparator, Text: line})
			continue
		}
		path, sep, num, text := m[1], m[2], m[3], m[4]
		it := byULID[ulidFromPath(path)]
		number, itemID := path, ""
		if it != nil {
			number, itemID = it.Number, it.ID
		}
		n, _ := strconv.Atoi(num)
		if sep == ":" {
			rows = append(rows, FindRow{Kind: RowMatch, ID: itemID, Number: number, Line: n, Text: text})
		} else {
			rows = append(rows, FindRow{Kind: RowContext, Number: number, Line: n, Text: text})
		}
	}
	return rows, nil
}

// findFallback is the no-rg path: a pure-Go regex scan over each ticket file's
// raw text, honoring -i and -w only (no -C context). It emits only match rows.
func (s *Store) findFallback(args FindArgs, items []*item.Item) ([]FindRow, error) {
	if args.Pattern == "" {
		return nil, userErr("find: a search pattern is required")
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
		return nil, userErr("find: invalid pattern: %v", err)
	}

	var rows []FindRow
	buf := make([]byte, 0, 64*1024)
	for _, it := range items {
		data, err := os.ReadFile(s.itemPath(it.ID))
		if err != nil {
			return nil, userErr("reading %s: %v", it.Number, err)
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
