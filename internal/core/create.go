package core

import (
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/editorx"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/id"
	"github.com/shivamshivanshu/kira/internal/ptr"
)

var subtypePattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

type CreateOpts struct {
	Type     string
	Subtype  string
	Title    string
	Body     *string
	Priority string
	Rank     string
	Owner    string
	Reporter string
	Labels   []string
	Parent   string
	Sprint   string
	Due      string
	Estimate *float64
	Fields   []FieldEdit
	NoEdit   bool
	FromFile string
	Force    bool
	Board    string
	Here     bool
	Blocking bool
}

// ValidateBlocking checks --blocking's precondition, shared by the CLI (for a
// fail-fast rejection before opening the store) and Store.Create (so the
// invariant holds for any caller, not just the CLI).
func (opts CreateOpts) ValidateBlocking() error {
	if opts.Blocking && !opts.Here {
		return errx.User("--blocking requires --here")
	}
	return nil
}

func (s *Store) ResolveTemplate(cfg *datamodel.Config, opts CreateOpts) (string, error) {
	if err := s.resolveOptsIdentities(cfg, &opts); err != nil {
		return "", err
	}
	base, err := s.templateDraft(opts.Type, opts.Subtype)
	if err != nil {
		return "", err
	}
	return serializeDraft(applyFlags(base, opts)), nil
}

// resolveOptsIdentities binds the @me token in the dedicated owner/reporter
// flags, which applyFlags copies verbatim; --field owner=@me takes the
// separate applyFieldEdits path, which resolves it there.
func (s *Store) resolveOptsIdentities(cfg *datamodel.Config, opts *CreateOpts) error {
	for _, field := range []*string{&opts.Owner, &opts.Reporter} {
		resolved, err := s.resolveMe(cfg, *field)
		if err != nil {
			return err
		}
		*field = resolved
	}
	return nil
}

func (s *Store) Create(cfg *datamodel.Config, opts CreateOpts) (*datamodel.CreateResult, error) {
	if err := opts.ValidateBlocking(); err != nil {
		return nil, err
	}
	if err := s.resolveOptsIdentities(cfg, &opts); err != nil {
		return nil, err
	}
	var hereActive string
	if opts.Here {
		active, err := s.resolveHere(cfg, &opts)
		if err != nil {
			return nil, err
		}
		hereActive = active
	}
	res, err := s.createLocked(cfg, opts)
	if err != nil {
		return nil, err
	}
	if opts.Blocking {
		if _, err := s.Link(cfg, hereActive, LinkOpts{Target: LinkBlockedBy, Ref: res.Number, Force: opts.Force}); err != nil {
			return res, errx.User("created %s, but marking it as a blocker of the active ticket failed: %v", res.Number, err).
				WithHint("the ticket exists; link it manually with `kira link <active-ticket> --blocked-by %s`", res.Number)
		}
	}
	return res, nil
}

func (s *Store) resolveHere(_ *datamodel.Config, opts *CreateOpts) (string, error) {
	ap, ok := s.readActive()
	if !ok {
		return "", errx.User("no active ticket").WithHint("start one with kira workon <id>")
	}
	items, _, err := s.LoadAll()
	if err != nil {
		return "", err
	}
	active, ok := datamodel.IndexByID(items)[ap.Ticket]
	if !ok {
		return "", errx.User("active ticket %s resolves to no item", ap.Ticket).WithHint("start one with kira workon <id>")
	}
	if opts.Parent == "" {
		if datamodel.IsContainer(active.Type) {
			opts.Parent = active.ID
		} else if active.Parent != nil {
			opts.Parent = *active.Parent
		}
	}
	if opts.Sprint == "" {
		opts.Sprint = ptr.Deref(active.Sprint)
	}
	return active.ID, nil
}

func (s *Store) createLocked(cfg *datamodel.Config, opts CreateOpts) (*datamodel.CreateResult, error) {
	wf, ok := datamodel.WorkflowOf(opts.Type)
	if !ok {
		return nil, UnknownTypeError(opts.Type)
	}

	boardKey, err := resolveBoardKey(cfg, opts.Board)
	if err != nil {
		return nil, err
	}

	base, err := s.templateDraft(opts.Type, opts.Subtype)
	if err != nil {
		return nil, err
	}
	base = applyFlags(base, opts)
	if opts.Here && !slices.Contains(base.Labels, datamodel.CapturedLabel) {
		base.Labels = append(base.Labels, datamodel.CapturedLabel)
	}

	d, err := s.draftForCreate(cfg, opts, boardKey, wf.Initial, base)
	if err != nil {
		return nil, err
	}

	release, err := s.fs().Lock()
	if err != nil {
		return nil, err
	}
	defer release()

	ld, err := s.load(cfg)
	if err != nil {
		return nil, err
	}

	sys := newSystemFields(ld.snap, boardKey, opts.Type, wf.Initial)
	finalItem := itemFromDraft(d, sys)
	if errs := s.applyFieldEdits(cfg, finalItem, opts.Fields); len(errs) > 0 {
		return nil, errx.Invalid(invalidItemPrefix, errs)
	}
	hard, warns := newValidationCtx(cfg, ld.items, ld.resolver, opts.Force).validateMutation(nil, finalItem)
	if len(hard) > 0 {
		return nil, errx.Invalid(invalidItemPrefix, hard)
	}

	path, err := s.fs().WriteItem(finalItem)
	if err != nil {
		return nil, err
	}
	subject := cfg.Commit.SubjectPrefix + finalItem.Number + " create " + quoteTitle(finalItem.Title)
	cs := &datamodel.ChangeSet{
		After:   finalItem,
		Paths:   []string{path},
		Subject: subject,
	}
	if err := s.commit(cfg, cs); err != nil {
		return nil, err
	}
	res := &datamodel.CreateResult{
		ID:         finalItem.ID,
		Number:     finalItem.Number,
		Board:      boardKey,
		Type:       finalItem.Type,
		Title:      finalItem.Title,
		State:      finalItem.State,
		Category:   categoryString(finalItem.Type, finalItem.State),
		Owner:      finalItem.Owner,
		Labels:     nonNil(finalItem.Labels),
		Parent:     finalItem.Parent,
		Priority:   finalItem.Priority,
		Resolution: finalItem.Resolution,
		Path:       path,
		// A file that would not parse is why a number can look skipped; say so
		// here rather than only on list.
		Warnings: append(literalWarnings(ld.warnings), warningsFromErrors(warns)...),
	}
	if finalItem.Parent != nil {
		if num, ok := parentNumberMap(ld.items)[*finalItem.Parent]; ok {
			res.ParentNumber = &num
		}
	}
	return res, nil
}

func (s *Store) draftForCreate(cfg *datamodel.Config, opts CreateOpts, boardKey, initialState string, base draft) (draft, error) {
	switch {
	case opts.FromFile != "":
		content, err := readSource(opts.FromFile)
		if err != nil {
			return draft{}, err
		}
		d, perr := parseDraft(stripErrorBanner(content))
		if perr != nil {
			return draft{}, errx.User("--from-file: %v", perr)
		}
		// The file plays the same role as the template it replaces: it supplies
		// the values, and an explicit flag still overrides them.
		return applyFlags(d, opts), nil
	case opts.NoEdit:
		return base, nil
	default:
		ld, err := s.load(cfg)
		if err != nil {
			return draft{}, err
		}
		sys := newSystemFields(ld.snap, boardKey, opts.Type, initialState)
		vctx := newValidationCtx(cfg, ld.items, ld.resolver, opts.Force)
		content, err := runEditor(cfg.UI.Editor, editorx.Stdio{}, serializeDraft(base), vctx.validateBuffer(func(c string) (*datamodel.Item, []error) {
			d, perr := parseDraft(c)
			if perr != nil {
				return nil, []error{perr}
			}
			return itemFromDraft(d, sys), nil
		}))
		if err != nil {
			return draft{}, err
		}
		d, _ := parseDraft(content)
		return d, nil
	}
}

type systemFields struct {
	ulid    string
	number  string
	typ     string
	state   string
	created string
}

func newSystemFields(snap id.Snapshot, boardKey, typ, state string) systemFields {
	u := id.Mint()
	return systemFields{
		ulid:    u.String(),
		number:  id.NewAllocator(snap, boardKey).Alloc(),
		typ:     typ,
		state:   state,
		created: time.Now().Format(time.RFC3339),
	}
}

func itemFromDraft(d draft, sys systemFields) *datamodel.Item {
	return &datamodel.Item{
		ID:        sys.ulid,
		Number:    sys.number,
		Aliases:   []string{},
		Type:      sys.typ,
		Subtype:   normPtr(d.Subtype),
		Title:     d.Title,
		State:     sys.state,
		Priority:  normPtr(d.Priority),
		Rank:      normPtr(d.Rank),
		Owner:     normPtr(d.Owner),
		Reporter:  normPtr(d.Reporter),
		Labels:    nonNil(d.Labels),
		Parent:    normPtr(d.Parent),
		BlockedBy: []string{},
		Sprint:    normPtr(d.Sprint),
		Due:       normPtr(d.Due),
		Estimate:  d.Estimate,
		Created:   sys.created,
		Updated:   sys.created,
		Body:      d.Body,
	}
}

func normPtr(p *string) *string {
	return ptr.NilIfEmpty(ptr.Deref(p))
}

func (s *Store) templateDraft(typ, subtype string) (draft, error) {
	if subtype != "" && !subtypePattern.MatchString(subtype) {
		return draft{}, errx.User("invalid subtype %q: allowed characters are letters, digits, '-' and '_'", subtype)
	}
	data, err := os.ReadFile(s.templatePath(typ, subtype))
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

func (s *Store) templatePath(typ, subtype string) string {
	dir := s.fs().TemplateDir()
	if subtype != "" {
		p := filepath.Join(dir, typ+"."+subtype+".md")
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return filepath.Join(dir, typ+".md")
}

func applyFlags(d draft, opts CreateOpts) draft {
	d.Type = opts.Type
	if opts.Body != nil {
		d.Body = normalizeEOL(*opts.Body)
	}
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
		d.Parent = &opts.Parent
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

func quoteTitle(title string) string {
	return `"` + strings.ReplaceAll(title, `"`, `\"`) + `"`
}
