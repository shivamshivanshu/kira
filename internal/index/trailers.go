package index

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"maps"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/gitx"
	"github.com/shivamshivanshu/kira/internal/id"
)

type Options struct {
	ProjectKey       string
	BoardKeys        []string
	TrailerKey       string
	CloseTrailer     string
	LandedRef        string
	Closes           bool
	LinkMarkers      []datamodel.LinkMarker
	ReferenceMarkers []datamodel.ReferenceMarker
}

type LinkKind string

const (
	LinkLinked     LinkKind = "linked"
	LinkReferenced LinkKind = "referenced"
)

type CommitLink struct {
	SHA     string
	Subject string
	Author  string
	Ts      string
	Kind    LinkKind
}

type CloseCandidate struct {
	ULID        string
	CommitterTs string
}

type CloseScan struct {
	LandedRef  string
	LandedHead string
	Candidates []CloseCandidate
	Unknown    []string
}

const trailerRef = "HEAD"

func (i *Index) scanTrailers(root gitx.Repo, opts Options, head string, prev meta, numbers map[string]string) ([]gitx.Commit, map[string]string, error) {
	wm := maps.Clone(prev.TrailerWatermarks)
	if wm == nil {
		wm = map[string]string{}
	}
	if opts.TrailerKey == "" || head == "" || wm[trailerRef] == head {
		return nil, wm, nil
	}

	rangeExpr, rewrite, err := trailerRange(root, wm[trailerRef], head)
	if err != nil {
		return nil, nil, err
	}
	commits, err := root.LogTrailers(rangeExpr, opts.TrailerKey, opts.CloseTrailer)
	if err != nil {
		return nil, nil, err
	}
	if err := i.upsertCommitLinks(commits, numbers, opts, rewrite); err != nil {
		return nil, nil, err
	}
	wm[trailerRef] = head
	return commits, wm, nil
}

func (i *Index) collectCloses(root gitx.Repo, opts Options, prev meta, numbers map[string]string, head string, headCommits []gitx.Commit) (CloseScan, error) {
	cs := CloseScan{LandedRef: opts.LandedRef}
	landedHead, err := root.Output("rev-parse", "--verify", "--quiet", opts.LandedRef)
	if err != nil || landedHead == "" {
		return cs, nil
	}
	cs.LandedHead = landedHead
	prevWm := prev.TrailerWatermarks[opts.LandedRef]
	if prevWm == landedHead {
		return cs, nil
	}

	commits := headCommits
	if commits == nil || landedHead != head || prevWm != prev.TrailerWatermarks[trailerRef] {
		rangeExpr, _, err := trailerRange(root, prevWm, landedHead)
		if err != nil {
			return CloseScan{}, err
		}
		commits, err = root.LogTrailers(rangeExpr, opts.TrailerKey, opts.CloseTrailer)
		if err != nil {
			return CloseScan{}, err
		}
	}

	cs.Candidates, cs.Unknown = latestCloses(commits, numbers)
	return cs, nil
}

func latestCloses(commits []gitx.Commit, numbers map[string]string) (candidates []CloseCandidate, unknown []string) {
	type acc struct {
		ts  time.Time
		raw string
	}
	latest := map[string]acc{}
	unknownSeen := map[string]bool{}
	var order []string
	for _, c := range commits {
		ct, _ := time.Parse(time.RFC3339, c.Timestamp)
		for _, value := range c.Closes {
			ulid, ok := numbers[strings.ToUpper(value)]
			if !ok {
				if !unknownSeen[value] {
					unknownSeen[value] = true
					unknown = append(unknown, value)
				}
				continue
			}
			cur, seen := latest[ulid]
			if !seen {
				order = append(order, ulid)
				latest[ulid] = acc{ct, c.Timestamp}
			} else if ct.After(cur.ts) {
				latest[ulid] = acc{ct, c.Timestamp}
			}
		}
	}
	for _, ulid := range order {
		candidates = append(candidates, CloseCandidate{ULID: ulid, CommitterTs: latest[ulid].raw})
	}
	return candidates, unknown
}

func PersistLandedWatermark(cacheDir, landedRef, landedHead string) error {
	m, ok := loadMetaAt(cacheDir)
	if !ok {
		return nil
	}
	if m.TrailerWatermarks == nil {
		m.TrailerWatermarks = map[string]string{}
	}
	m.TrailerWatermarks[landedRef] = landedHead
	return saveMetaAt(cacheDir, m)
}

func trailerRange(root gitx.Repo, watermark, head string) (rangeExpr string, rewrite bool, err error) {
	if watermark == "" {
		return head, true, nil
	}
	ancestor, err := root.IsAncestor(gitx.Ancestor(watermark), gitx.Descendant(head))
	if err != nil {
		return "", false, err
	}
	if !ancestor {
		return head, true, nil
	}
	return watermark + ".." + head, false, nil
}

type linkPolicy struct {
	trailer bool
	subject bool
	bare    bool
	marker  *regexp.Regexp
	bareRef *regexp.Regexp
}

func scanConfigHash(opts Options) string {
	var b strings.Builder
	b.WriteString(opts.ProjectKey)
	b.WriteByte(0)
	for _, k := range opts.BoardKeys {
		b.WriteString(k)
		b.WriteByte(0x1f)
	}
	b.WriteByte(0)
	b.WriteString(opts.TrailerKey)
	b.WriteByte(0)
	b.WriteString(opts.CloseTrailer)
	b.WriteByte(0)
	for _, m := range opts.LinkMarkers {
		b.WriteString(string(m))
		b.WriteByte(0x1f)
	}
	b.WriteByte(0)
	for _, m := range opts.ReferenceMarkers {
		b.WriteString(string(m))
		b.WriteByte(0x1f)
	}
	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:])
}

func newLinkPolicy(opts Options, numbers map[string]string) linkPolicy {
	return linkPolicy{
		trailer: slices.Contains(opts.LinkMarkers, datamodel.LinkMarkerTrailer),
		subject: slices.Contains(opts.LinkMarkers, datamodel.LinkMarkerSubject),
		bare:    slices.Contains(opts.ReferenceMarkers, datamodel.ReferenceMarkerBare),
		marker:  subjectMarkerPattern(opts.ProjectKey),
		bareRef: lenientPattern(opts.BoardKeys, numbers),
	}
}

func (i *Index) upsertCommitLinks(commits []gitx.Commit, numbers map[string]string, opts Options, rewrite bool) error {
	tx, err := i.db.Begin()
	if err != nil {
		return errx.User("beginning commit-link tx: %v", err)
	}
	defer tx.Rollback()
	if rewrite {
		if _, err := tx.Exec("DELETE FROM commit_links"); err != nil {
			return errx.User("clearing commit links: %v", err)
		}
	}
	insLinked, err := tx.Prepare(`INSERT OR REPLACE INTO commit_links
		(item_id, sha, subject, author, ts, kind) VALUES (?,?,?,?,?,?)`)
	if err != nil {
		return errx.User("preparing commit-link insert: %v", err)
	}
	defer insLinked.Close()
	insReferenced, err := tx.Prepare(`INSERT OR IGNORE INTO commit_links
		(item_id, sha, subject, author, ts, kind) VALUES (?,?,?,?,?,?)`)
	if err != nil {
		return errx.User("preparing commit-link insert: %v", err)
	}
	defer insReferenced.Close()

	pol := newLinkPolicy(opts, numbers)
	for _, c := range commits {
		linked, referenced := resolveItemRefs(c, numbers, pol)
		for _, ulid := range linked {
			if _, err := insLinked.Exec(ulid, c.SHA, c.Subject, c.Author, c.Timestamp, LinkLinked); err != nil {
				return errx.User("inserting commit link: %v", err)
			}
		}
		for _, ulid := range referenced {
			if _, err := insReferenced.Exec(ulid, c.SHA, c.Subject, c.Author, c.Timestamp, LinkReferenced); err != nil {
				return errx.User("inserting commit link: %v", err)
			}
		}
	}
	return commit(tx)
}

func resolveItemRefs(c gitx.Commit, numbers map[string]string, pol linkPolicy) (linked, referenced []string) {
	linkedSeen := map[string]bool{}
	addLinked := func(token string) {
		if ulid, ok := numbers[strings.ToUpper(token)]; ok && !linkedSeen[ulid] {
			linkedSeen[ulid] = true
			linked = append(linked, ulid)
		}
	}
	if pol.trailer {
		for _, t := range c.Tickets {
			addLinked(t)
		}
	}
	if pol.subject && pol.marker != nil && len(linked) == 0 {
		if m := pol.marker.FindString(c.Subject); m != "" {
			addLinked(strings.Trim(m, "[]"))
		}
	}

	refSeen := map[string]bool{}
	if pol.bare && pol.bareRef != nil {
		for _, tok := range pol.bareRef.FindAllString(stripMarkers(bodyOutsideTrailers(c), pol.marker), -1) {
			if ulid, ok := numbers[strings.ToUpper(tok)]; ok && !linkedSeen[ulid] && !refSeen[ulid] {
				refSeen[ulid] = true
				referenced = append(referenced, ulid)
			}
		}
	}
	return linked, referenced
}

func stripMarkers(text string, marker *regexp.Regexp) string {
	if marker == nil {
		return text
	}
	return marker.ReplaceAllStringFunc(text, func(m string) string {
		return strings.Trim(m, "[]")
	})
}

func bodyOutsideTrailers(c gitx.Commit) string {
	block := strings.TrimRight(c.TrailerBlock, "\n")
	if block == "" {
		return c.Body
	}
	if idx := strings.LastIndex(c.Body, block); idx >= 0 {
		return c.Body[:idx]
	}
	return c.Body
}

func lenientPattern(boardKeys []string, numbers map[string]string) *regexp.Regexp {
	seen := map[string]bool{}
	var prefixes []string
	add := func(p string) {
		up := strings.ToUpper(p)
		if up != "" && !seen[up] {
			seen[up] = true
			prefixes = append(prefixes, up)
		}
	}
	for _, k := range boardKeys {
		add(k)
	}
	for full := range numbers {
		add(id.KeyOf(full))
	}
	if len(prefixes) == 0 {
		return nil
	}
	slices.SortFunc(prefixes, func(a, b string) int {
		if len(a) != len(b) {
			return len(b) - len(a)
		}
		return strings.Compare(a, b)
	})
	quoted := make([]string, len(prefixes))
	for i, p := range prefixes {
		quoted[i] = regexp.QuoteMeta(p)
	}
	return regexp.MustCompile(`(?i)\b(?:` + strings.Join(quoted, "|") + `)-\d+\b`)
}

func subjectMarkerPattern(projectKey string) *regexp.Regexp {
	if projectKey == "" {
		return nil
	}
	return regexp.MustCompile(`\[\[` + regexp.QuoteMeta(projectKey) + `-\d+\]\]`)
}

func (i *Index) numberToULID() (map[string]string, error) {
	numbers := map[string]string{}
	for _, q := range []string{"SELECT number, id FROM items", "SELECT number, item_id FROM aliases"} {
		rows, err := i.db.Query(q)
		if err != nil {
			return nil, errx.User("querying numbers: %v", err)
		}
		if err := eachPair(rows, func(r *sql.Rows) error {
			var number, ulid string
			if err := r.Scan(&number, &ulid); err != nil {
				return errx.User("scanning number: %v", err)
			}
			numbers[strings.ToUpper(number)] = ulid
			return nil
		}); err != nil {
			return nil, err
		}
	}
	return numbers, nil
}

func (i *Index) CommitLinks(itemID string) ([]CommitLink, error) {
	rows, err := i.db.Query(`SELECT sha, subject, author, ts, kind FROM commit_links
		WHERE item_id = ? ORDER BY kind = ? DESC, ts DESC, rowid`, itemID, LinkLinked)
	if err != nil {
		return nil, errx.User("querying commit links: %v", err)
	}
	defer rows.Close()
	var links []CommitLink
	for rows.Next() {
		var l CommitLink
		if err := rows.Scan(&l.SHA, &l.Subject, &l.Author, &l.Ts, &l.Kind); err != nil {
			return nil, errx.User("scanning commit link: %v", err)
		}
		links = append(links, l)
	}
	return links, rows.Err()
}
