// Package doctor runs read-only consistency checks over a kira repo — id
// collisions, dangling refs, schema/state/vocab violations, epic cycles, index
// freshness, and hook/binary presence — and reports structured findings that a
// future fixer can consume. It performs no writes.
package doctor

import (
	"path/filepath"
	"slices"
	"strings"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/id"
	"github.com/shivamshivanshu/kira/internal/storage"
)

type File struct {
	Path    string
	Content string
}

type parsedFile struct {
	path   string
	item   *datamodel.Item
	parsed bool
	lint   []Finding
}

func (s parsedFile) identified() bool { return s.item != nil && s.item.ID != "" }

func scan(files []File) []parsedFile {
	out := make([]parsedFile, len(files))
	for i, f := range files {
		it, parsed, lf := Lint(f.Content)
		out[i] = parsedFile{path: f.Path, item: it, parsed: parsed, lint: lf}
	}
	return out
}

func Run(cfg *datamodel.Config, files []File, strays []string, env Env) *Report {
	files = slices.Clone(files)
	slices.SortFunc(files, func(a, b File) int { return strings.Compare(a.Path, b.Path) })
	scanned := scan(files)

	var items []*datamodel.Item
	pathByID := map[string]string{}
	for _, s := range scanned {
		if s.identified() {
			items = append(items, s.item)
			pathByID[s.item.ID] = s.path
		}
	}

	resolver := resolverFor(cfg, items)
	var findings []Finding
	for _, s := range scanned {
		findings = append(findings, fileFindings(cfg, resolver, s, true)...)
	}
	findings = append(findings, stampByID(Collisions(items), pathByID)...)
	findings = append(findings, stampByID(SequentialOutliers(items), pathByID)...)
	findings = append(findings, stampByID(EpicCycles(items, resolver), pathByID)...)
	findings = append(findings, stampByID(NonEpicParents(items, resolver), pathByID)...)
	findings = append(findings, stampByID(RefCycles(items, resolver), pathByID)...)
	findings = append(findings, strayFindings(strays)...)
	findings = append(findings, envFindings(env)...)

	orderFindings(findings)
	return newReport(findings)
}

func strayFindings(strays []string) []Finding {
	out := make([]Finding, len(strays))
	for i, name := range strays {
		out[i] = Finding{
			Class:    ClassStray,
			Severity: SeverityError,
			Path:     name,
			Message:  storage.StrayMessage,
		}
	}
	return out
}

func Validate(cfg *datamodel.Config, storeFiles, targets []File) *Report {
	targetScan := scan(targets)
	resolver := resolverFor(cfg, dedupItems(scan(storeFiles), targetScan))

	var findings []Finding
	for _, s := range targetScan {
		findings = append(findings, fileFindings(cfg, resolver, s, false)...)
	}
	orderFindings(findings)
	return newReport(findings)
}

func fileFindings(cfg *datamodel.Config, resolver *id.Resolver, s parsedFile, checkIdentity bool) []Finding {
	findings := stamp(slices.Clone(s.lint), s.path, s.item)
	if checkIdentity && s.identified() {
		findings = append(findings, stamp(identityFindings(s.path, s.item), s.path, s.item)...)
	}
	if s.parsed {
		findings = append(findings, stamp(Check(cfg, resolver, s.item), s.path, s.item)...)
	}
	return findings
}

func dedupItems(scans ...[]parsedFile) []*datamodel.Item {
	seen := map[string]bool{}
	var items []*datamodel.Item
	for _, sc := range scans {
		for _, s := range sc {
			if s.identified() && !seen[s.item.ID] {
				seen[s.item.ID] = true
				items = append(items, s.item)
			}
		}
	}
	return items
}

func identityFindings(path string, it *datamodel.Item) []Finding {
	stem := strings.TrimSuffix(filepath.Base(path), ".md")
	if stem == "" || it.ID == stem {
		return nil
	}
	return []Finding{{
		Class:    ClassSchema,
		Severity: SeverityError,
		Field:    datamodel.KeyID,
		Message:  "frontmatter id " + it.ID + " does not match filename stem " + stem,
	}}
}

func resolverFor(cfg *datamodel.Config, items []*datamodel.Item) *id.Resolver {
	return id.NewResolver(storage.Snapshot(cfg.Project.Key, items))
}

func stampByID(findings []Finding, pathByID map[string]string) []Finding {
	for i := range findings {
		if findings[i].Path == "" {
			findings[i].Path = pathByID[findings[i].ItemID]
		}
	}
	return findings
}

func orderFindings(findings []Finding) {
	slices.SortStableFunc(findings, func(a, b Finding) int {
		if c := comparePath(a.Path, b.Path); c != 0 {
			return c
		}
		if c := strings.Compare(string(a.Class), string(b.Class)); c != 0 {
			return c
		}
		if c := strings.Compare(a.Field, b.Field); c != 0 {
			return c
		}
		return strings.Compare(a.Message, b.Message)
	})
}

func comparePath(a, b string) int {
	switch {
	case a == b:
		return 0
	case a == "":
		return 1
	case b == "":
		return -1
	default:
		return strings.Compare(a, b)
	}
}
