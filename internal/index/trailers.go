package index

import (
	"regexp"
	"strings"
	"time"

	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/gitx"
)

type Options struct {
	ProjectKey   string
	TrailerKey   string
	CloseTrailer string
	LandedRef    string
	Closes       bool
}

type CommitLink struct {
	SHA     string
	Subject string
	Author  string
	Ts      string
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

const (
	trailerRef    = "HEAD"
	sourceTrailer = "trailer"
	sourceLenient = "lenient"
)

func (i *Index) scanTrailers(root gitx.Repo, opts Options, head string, prev meta, numbers map[string]string) ([]gitx.Commit, map[string]string, error) {
	wm := cloneWatermarks(prev.TrailerWatermarks)
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
	if err := i.upsertCommitLinks(commits, numbers, opts.ProjectKey, rewrite); err != nil {
		return nil, nil, err
	}
	wm[trailerRef] = head
	return commits, wm, nil
}

func (i *Index) collectCloses(root gitx.Repo, opts Options, prev meta, numbers map[string]string, head string, landedRef string, headCommits []gitx.Commit) (CloseScan, error) {
	cs := CloseScan{LandedRef: landedRef}
	landedHead, err := root.Output("rev-parse", "--verify", "--quiet", landedRef)
	if err != nil || landedHead == "" {
		return cs, nil
	}
	cs.LandedHead = landedHead
	prevWm := prev.TrailerWatermarks[landedRef]
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
					cs.Unknown = append(cs.Unknown, value)
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
		cs.Candidates = append(cs.Candidates, CloseCandidate{ULID: ulid, CommitterTs: latest[ulid].raw})
	}
	return cs, nil
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
	ancestor, err := root.IsAncestor(watermark, head)
	if err != nil {
		return "", false, err
	}
	if !ancestor {
		return head, true, nil
	}
	return watermark + ".." + head, false, nil
}

func (i *Index) upsertCommitLinks(commits []gitx.Commit, numbers map[string]string, projectKey string, rewrite bool) error {
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
	insTrailer, err := tx.Prepare(`INSERT OR REPLACE INTO commit_links
		(item_id, sha, subject, author, ts, source) VALUES (?,?,?,?,?,?)`)
	if err != nil {
		return errx.User("preparing commit-link insert: %v", err)
	}
	defer insTrailer.Close()
	insLenient, err := tx.Prepare(`INSERT OR IGNORE INTO commit_links
		(item_id, sha, subject, author, ts, source) VALUES (?,?,?,?,?,?)`)
	if err != nil {
		return errx.User("preparing commit-link insert: %v", err)
	}
	defer insLenient.Close()

	lenient := lenientPattern(projectKey)
	for _, c := range commits {
		trailer, lenientHits := resolveItemRefs(c, numbers, lenient)
		for _, ulid := range trailer {
			if _, err := insTrailer.Exec(ulid, c.SHA, c.Subject, c.Author, c.Timestamp, sourceTrailer); err != nil {
				return errx.User("inserting commit link: %v", err)
			}
		}
		for _, ulid := range lenientHits {
			if _, err := insLenient.Exec(ulid, c.SHA, c.Subject, c.Author, c.Timestamp, sourceLenient); err != nil {
				return errx.User("inserting commit link: %v", err)
			}
		}
	}
	return commit(tx)
}

func resolveItemRefs(c gitx.Commit, numbers map[string]string, lenient *regexp.Regexp) (trailer, lenientHits []string) {
	seen := map[string]bool{}
	add := func(dst *[]string, token string) {
		if ulid, ok := numbers[strings.ToUpper(token)]; ok && !seen[ulid] {
			seen[ulid] = true
			*dst = append(*dst, ulid)
		}
	}
	for _, t := range c.Tickets {
		add(&trailer, t)
	}
	if lenient != nil {
		for _, tok := range lenient.FindAllString(bodyOutsideTrailers(c), -1) {
			add(&lenientHits, tok)
		}
	}
	return trailer, lenientHits
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

func lenientPattern(projectKey string) *regexp.Regexp {
	if projectKey == "" {
		return nil
	}
	return regexp.MustCompile(`\b` + regexp.QuoteMeta(projectKey) + `-\d+\b`)
}

func (i *Index) numberToULID() (map[string]string, error) {
	numbers := map[string]string{}
	for _, q := range []string{"SELECT number, id FROM items", "SELECT number, item_id FROM aliases"} {
		rows, err := i.db.Query(q)
		if err != nil {
			return nil, errx.User("querying numbers: %v", err)
		}
		for rows.Next() {
			var number, ulid string
			if err := rows.Scan(&number, &ulid); err != nil {
				rows.Close()
				return nil, errx.User("scanning number: %v", err)
			}
			numbers[strings.ToUpper(number)] = ulid
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, err
		}
		rows.Close()
	}
	return numbers, nil
}

func (i *Index) CommitLinks(itemID string) ([]CommitLink, error) {
	rows, err := i.db.Query(`SELECT sha, subject, author, ts FROM commit_links
		WHERE item_id = ? ORDER BY source = ? DESC, ts DESC, rowid`, itemID, sourceTrailer)
	if err != nil {
		return nil, errx.User("querying commit links: %v", err)
	}
	defer rows.Close()
	var links []CommitLink
	for rows.Next() {
		var l CommitLink
		if err := rows.Scan(&l.SHA, &l.Subject, &l.Author, &l.Ts); err != nil {
			return nil, errx.User("scanning commit link: %v", err)
		}
		links = append(links, l)
	}
	return links, rows.Err()
}

func cloneWatermarks(src map[string]string) map[string]string {
	dst := make(map[string]string, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
