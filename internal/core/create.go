package core

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/id"
)

type CreateOpts struct {
	Type     string
	Subtype  string
	Title    string
	Priority string
	Rank     string
	Owner    string
	Reporter string
	Labels   []string
	Parent   string
	Sprint   string
	Due      string
	Estimate *float64
	NoEdit   bool
	FromFile string
	Force    bool
}

func (s *Store) ResolveTemplate(opts CreateOpts) (string, error) {
	base, err := s.templateDraft(opts.Type)
	if err != nil {
		return "", err
	}
	return serializeDraft(applyFlags(base, opts)), nil
}

func (s *Store) Create(cfg *datamodel.Config, opts CreateOpts) (*datamodel.CreateResult, error) {
	wf, ok := cfg.Workflows[opts.Type]
	if !ok {
		return nil, errx.User("no workflow configured for type %q", opts.Type)
	}

	release, err := s.fs().Lock()
	if err != nil {
		return nil, err
	}
	defer release()

	_, snap, resolver, err := s.load(cfg)
	if err != nil {
		return nil, err
	}

	u := id.Mint()
	sys := systemFields{
		ulid:    u.String(),
		number:  allocateNumber(cfg, snap, u),
		typ:     opts.Type,
		state:   wf.Initial,
		created: time.Now().Format(time.RFC3339),
	}

	assemble := func(d draft) (*datamodel.Item, []error, []error) {
		it := itemFromDraft(d, sys)
		hard, warns := validateAssembled(cfg, it, resolver, opts.Force)
		return it, hard, warns
	}

	base, err := s.templateDraft(opts.Type)
	if err != nil {
		return nil, err
	}
	base = applyFlags(base, opts)

	var finalItem *datamodel.Item
	var warns []error
	switch {
	case opts.FromFile != "":
		content, err := readSource(opts.FromFile)
		if err != nil {
			return nil, err
		}
		d, perr := parseDraft(stripErrorBanner(content))
		if perr != nil {
			return nil, errx.User("--from-file: %v", perr)
		}
		it, errs, w := assemble(d)
		if len(errs) > 0 {
			return nil, errx.Invalid(errs)
		}
		finalItem, warns = it, w
	case opts.NoEdit:
		it, errs, w := assemble(base)
		if len(errs) > 0 {
			return nil, errx.Invalid(errs)
		}
		finalItem, warns = it, w
	default:
		content, err := runEditor(serializeDraft(base), func(c string) []error {
			d, perr := parseDraft(c)
			if perr != nil {
				return []error{perr}
			}
			_, errs, _ := assemble(d)
			return errs
		})
		if err != nil {
			return nil, err
		}
		d, _ := parseDraft(content)
		finalItem, _, warns = assemble(d)
	}

	emitWarnings(warns)

	path, err := s.writeItem(finalItem)
	if err != nil {
		return nil, err
	}
	subject := "kira: create " + finalItem.Number + " " + quoteTitle(finalItem.Title)
	if err := s.finalize(cfg.Commit.Mode, cfg.Commit.Trailer, subject, finalItem.Number, path); err != nil {
		return nil, err
	}
	return &datamodel.CreateResult{
		ID:     finalItem.ID,
		Number: finalItem.Number,
		Type:   finalItem.Type,
		Title:  finalItem.Title,
		State:  finalItem.State,
		Path:   path,
	}, nil
}

type systemFields struct {
	ulid    string
	number  string
	typ     string
	state   string
	created string
}

func itemFromDraft(d draft, sys systemFields) *datamodel.Item {
	return &datamodel.Item{
		ID:        sys.ulid,
		Number:    sys.number,
		Aliases:   []string{},
		Type:      sys.typ,
		Subtype:   nonEmptyPtr(d.Subtype),
		Title:     d.Title,
		State:     sys.state,
		Priority:  nonEmptyPtr(d.Priority),
		Rank:      nonEmptyPtr(d.Rank),
		Owner:     nonEmptyPtr(d.Owner),
		Reporter:  nonEmptyPtr(d.Reporter),
		Labels:    nonNil(d.Labels),
		Epic:      nonEmptyPtr(d.Epic),
		BlockedBy: []string{},
		Sprint:    nonEmptyPtr(d.Sprint),
		Due:       nonEmptyPtr(d.Due),
		Estimate:  d.Estimate,
		Created:   sys.created,
		Updated:   sys.created,
		Body:      d.Body,
	}
}

func (s *Store) templateDraft(typ string) (draft, error) {
	data, err := os.ReadFile(s.templatePath(typ))
	if err != nil {
		if os.IsNotExist(err) {
			d, _ := parseDraft(defaultTemplate(typ))
			return d, nil
		}
		return draft{}, errx.User("reading template: %v", err)
	}
	d, perr := parseDraft(string(data))
	if perr != nil {
		return draft{}, errx.User("template %s.md: %v", typ, perr)
	}
	d.Type = typ
	return d, nil
}

func (s *Store) templatePath(typ string) string {
	return filepath.Join(s.fs().TemplateDir(), typ+".md")
}

func applyFlags(d draft, opts CreateOpts) draft {
	d.Type = opts.Type
	if opts.Title != "" {
		d.Title = opts.Title
	}
	if opts.Subtype != "" {
		d.Subtype = &opts.Subtype
	}
	if opts.Priority != "" {
		d.Priority = &opts.Priority
	}
	if opts.Rank != "" {
		d.Rank = &opts.Rank
	}
	if opts.Owner != "" {
		d.Owner = &opts.Owner
	}
	if opts.Reporter != "" {
		d.Reporter = &opts.Reporter
	}
	if len(opts.Labels) > 0 {
		d.Labels = opts.Labels
	}
	if opts.Parent != "" {
		d.Epic = &opts.Parent
	}
	if opts.Sprint != "" {
		d.Sprint = &opts.Sprint
	}
	if opts.Due != "" {
		d.Due = &opts.Due
	}
	if opts.Estimate != nil {
		d.Estimate = opts.Estimate
	}
	return d
}

func allocateNumber(cfg *datamodel.Config, snap id.Snapshot, u id.ULID) string {
	if cfg.ID.Style == datamodel.IDStyleHash {
		return id.HashNumber(cfg.Project.Key, u)
	}
	return id.Allocate(snap).String()
}

func readSource(src string) (string, error) {
	if src == "-" {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", errx.User("reading stdin: %v", err)
		}
		return string(data), nil
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return "", errx.User("reading %s: %v", src, err)
	}
	return string(data), nil
}

func nonEmptyPtr(p *string) *string {
	if p == nil || *p == "" {
		return nil
	}
	return p
}

func quoteTitle(title string) string {
	return `"` + strings.ReplaceAll(title, `"`, `\"`) + `"`
}
