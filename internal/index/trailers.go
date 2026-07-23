package index

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"maps"
	"regexp"
	"slices"
	"strings"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/gitx"
	"github.com/shivamshivanshu/kira/internal/id"
	"github.com/shivamshivanshu/kira/internal/setx"
	"github.com/shivamshivanshu/kira/internal/timex"
)

// Options configures how commit trailers and subjects are scanned for ticket
// links, references, and closes.
type Options struct {
	ProjectKey       string
	BoardKeys        []string
	TrailerKey       string
	CloseTrailer     string
	SubjectPrefix    string
	LandedRef        string
	Closes           bool
	LinkMarkers      []datamodel.LinkMarker
	ReferenceMarkers []datamodel.ReferenceMarker
}

// LinkKind distinguishes a commit that links an item (via trailer, subject
// marker, or leading number) from one that merely references it.
type LinkKind string

// LinkLinked and LinkReferenced are the two kinds of commit-to-item association.
const (
	LinkLinked     LinkKind = "linked"
	LinkReferenced LinkKind = "referenced"
)

// CommitLink is a single commit associated with an item via trailer or
// reference scanning.
type CommitLink struct {
	SHA     string
	Subject string
	Author  string
	Ts      string
	Kind    LinkKind
}

// CloseCandidate is an item eligible to be closed by a landed commit.
type CloseCandidate struct {
	ULID        string
	CommitterTs string
}

// CloseScan is the result of scanning landed commits for items to close.
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
	latest := map[string]string{}
	unknownSeen := map[string]bool{}
	var order []string
	for _, c := range commits {
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
				latest[ulid] = c.Timestamp
			} else if closesLater(c.Timestamp, cur) {
				latest[ulid] = c.Timestamp
			}
		}
	}
	for _, ulid := range order {
		candidates = append(candidates, CloseCandidate{ULID: ulid, CommitterTs: latest[ulid]})
	}
	return candidates, unknown
}

// closesLater reports whether a supersedes b as the latest close timestamp for
// an item. A timestamp that fails to parse can never win the race, but it can
// always be superseded by one that does — an unparseable entry must not get
// stuck as "latest" forever.
func closesLater(a, b string) bool {
	cmp, aOK, bOK := timex.CompareRFC3339(a, b)
	if aOK && bOK {
		return cmp > 0
	}
	return aOK && !bOK
}

// PersistLandedWatermark records landedHead as the last-scanned commit for
// landedRef, so the next scan for that ref starts from it.
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
	trailer       bool
	subject       bool
	leadingNumber bool
	bare          bool
	subjectPrefix string
	marker        *regexp.Regexp
	bareRef       *regexp.Regexp
}

const scanPolicyVersion = "2"

func scanConfigHash(opts Options) string {
	var b strings.Builder
	b.WriteString(scanPolicyVersion)
	b.WriteByte(0)
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
	b.WriteByte(0)
	b.WriteString(opts.SubjectPrefix)
	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:])
}

func newLinkPolicy(opts Options, numbers map[string]string) linkPolicy {
	return linkPolicy{
		trailer:       slices.Contains(opts.LinkMarkers, datamodel.LinkMarkerTrailer),
		subject:       slices.Contains(opts.LinkMarkers, datamodel.LinkMarkerSubject),
		leadingNumber: slices.Contains(opts.LinkMarkers, datamodel.LinkMarkerLeadingNumber),
		bare:          slices.Contains(opts.ReferenceMarkers, datamodel.ReferenceMarkerBare),
		subjectPrefix: opts.SubjectPrefix,
		marker:        subjectMarkerPattern(opts, numbers),
		bareRef:       lenientPattern(opts.BoardKeys, numbers),
	}
}

func (i *Index) upsertCommitLinks(commits []gitx.Commit, numbers map[string]string, opts Options, rewrite bool) error {
	tx, err := i.db.Begin()
	if err != nil {
		return errx.Env("beginning commit-link tx: %v", err)
	}
	defer func() { _ = tx.Rollback() }()
	if rewrite {
		if _, err := tx.Exec("DELETE FROM commit_links"); err != nil {
			return errx.Env("clearing commit links: %v", err)
		}
	}
	insLinked, err := tx.Prepare(`INSERT OR REPLACE INTO commit_links
		(item_id, sha, subject, author, ts, kind) VALUES (?,?,?,?,?,?)`)
	if err != nil {
		return errx.Env("preparing commit-link insert: %v", err)
	}
	defer func() { _ = insLinked.Close() }()
	insReferenced, err := tx.Prepare(`INSERT OR IGNORE INTO commit_links
		(item_id, sha, subject, author, ts, kind) VALUES (?,?,?,?,?,?)`)
	if err != nil {
		return errx.Env("preparing commit-link insert: %v", err)
	}
	defer func() { _ = insReferenced.Close() }()

	pol := newLinkPolicy(opts, numbers)
	for _, c := range commits {
		linked, referenced := resolveItemRefs(c, numbers, pol)
		for _, ulid := range linked {
			if _, err := insLinked.Exec(ulid, c.SHA, c.Subject, c.Author, c.Timestamp, LinkLinked); err != nil {
				return errx.Env("inserting commit link: %v", err)
			}
		}
		for _, ulid := range referenced {
			if _, err := insReferenced.Exec(ulid, c.SHA, c.Subject, c.Author, c.Timestamp, LinkReferenced); err != nil {
				return errx.Env("inserting commit link: %v", err)
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
	if pol.leadingNumber && pol.bareRef != nil && len(linked) == 0 && (!pol.trailer || len(c.Tickets) == 0) {
		s := strings.TrimLeft(strings.TrimPrefix(c.Subject, pol.subjectPrefix), " \t")
		if loc := pol.bareRef.FindStringIndex(s); loc != nil && loc[0] == 0 {
			addLinked(s[:loc[1]])
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

func keyPrefixes(keys []string, numbers map[string]string) []string {
	dedup := setx.NewDeduper[string]()
	var prefixes []string
	add := func(p string) {
		up := strings.ToUpper(p)
		if up != "" && dedup.Add(up) {
			prefixes = append(prefixes, up)
		}
	}
	for _, k := range keys {
		add(k)
	}
	for full := range numbers {
		add(id.KeyOf(full))
	}
	slices.SortFunc(prefixes, func(a, b string) int {
		if len(a) != len(b) {
			return len(b) - len(a)
		}
		return strings.Compare(a, b)
	})
	return prefixes
}

func prefixAlternation(prefixes []string) string {
	quoted := make([]string, len(prefixes))
	for i, p := range prefixes {
		quoted[i] = regexp.QuoteMeta(p)
	}
	return strings.Join(quoted, "|")
}

func lenientPattern(boardKeys []string, numbers map[string]string) *regexp.Regexp {
	prefixes := keyPrefixes(boardKeys, numbers)
	if len(prefixes) == 0 {
		return nil
	}
	return regexp.MustCompile(`(?i)\b(?:` + prefixAlternation(prefixes) + `)-\d+\b`)
}

func subjectMarkerPattern(opts Options, numbers map[string]string) *regexp.Regexp {
	prefixes := keyPrefixes(append([]string{opts.ProjectKey}, opts.BoardKeys...), numbers)
	if len(prefixes) == 0 {
		return nil
	}
	return regexp.MustCompile(`(?i)\[\[(?:` + prefixAlternation(prefixes) + `)-\d+\]\]`)
}

func (i *Index) numberToULID() (map[string]string, error) {
	numbers := map[string]string{}
	for _, q := range []string{"SELECT number, id FROM items", "SELECT number, item_id FROM aliases"} {
		rows, err := i.db.Query(q)
		if err != nil {
			return nil, errx.Env("querying numbers: %v", err)
		}
		if err := eachPair(rows, func(r *sql.Rows) error {
			var number, ulid string
			if err := r.Scan(&number, &ulid); err != nil {
				return errx.Env("scanning number: %v", err)
			}
			numbers[strings.ToUpper(number)] = ulid
			return nil
		}); err != nil {
			return nil, err
		}
	}
	return numbers, nil
}

// CommitLinks returns the commits linked to or referencing itemID, linked
// commits first and each group newest first.
func (i *Index) CommitLinks(itemID string) ([]CommitLink, error) {
	rows, err := i.db.Query(`SELECT sha, subject, author, ts, kind FROM commit_links
		WHERE item_id = ? ORDER BY kind = ? DESC, ts DESC, rowid`, itemID, LinkLinked)
	if err != nil {
		return nil, errx.Env("querying commit links: %v", err)
	}
	defer func() { _ = rows.Close() }()
	var links []CommitLink
	for rows.Next() {
		var l CommitLink
		if err := rows.Scan(&l.SHA, &l.Subject, &l.Author, &l.Ts, &l.Kind); err != nil {
			return nil, errx.Env("scanning commit link: %v", err)
		}
		links = append(links, l)
	}
	return links, rows.Err()
}
