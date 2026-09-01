package core

import (
	"cmp"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"

	"github.com/shivamshivanshu/kira/internal/config"
	"github.com/shivamshivanshu/kira/internal/datamodel"
)

// Explain renders the repo's live effective config as human-readable rules,
// each tagged with the tier (default | repo | user) that last set it.
func Explain(cfg *datamodel.Config) *datamodel.ExplainResult {
	def := config.Default()
	sections := []datamodel.ExplainSection{explainIdentity(cfg, def)}
	for _, typ := range slices.Sorted(maps.Keys(cfg.Workflows)) {
		sections = append(sections, explainWorkflow(cfg, def, typ))
	}
	sections = append(sections,
		explainCommit(cfg, def),
		explainVocab(cfg, def),
		explainUI(cfg, def),
		explainWorkon(cfg, def),
		explainMerge(cfg, def),
		explainSync(cfg, def),
		explainGit(cfg, def),
		explainEstimate(cfg, def),
		explainFilters(cfg, def),
		explainAutomation(cfg, def),
	)
	return &datamodel.ExplainResult{Sections: sections}
}

func explainGit(cfg, def *datamodel.Config) datamodel.ExplainSection {
	lines := []string{
		"landed_ref: " + orUnset(cfg.Git.LandedRef),
		"scan_since: " + orUnset(cfg.Git.ScanSince),
	}
	return datamodel.ExplainSection{Name: "git", Provenance: tierOf(changed(cfg.Git, def.Git), datamodel.ProvenanceRepo), Lines: lines}
}

func explainEstimate(cfg, def *datamodel.Config) datamodel.ExplainSection {
	lines := []string{"unit: " + string(cfg.Estimate.Unit)}
	return datamodel.ExplainSection{Name: "estimate", Provenance: tierOf(changed(cfg.Estimate, def.Estimate), datamodel.ProvenanceRepo), Lines: lines}
}

func explainFilters(cfg, def *datamodel.Config) datamodel.ExplainSection {
	lines := make([]string, 0, len(cfg.Filters))
	for _, name := range slices.Sorted(maps.Keys(cfg.Filters)) {
		lines = append(lines, name+": "+cfg.Filters[name])
	}
	if len(lines) == 0 {
		lines = []string{"none defined"}
	}
	return datamodel.ExplainSection{Name: "filters", Provenance: tierOf(changed(cfg.Filters, def.Filters), datamodel.ProvenanceRepo), Lines: lines}
}

// Automation hooks run shell commands, so the explain output names what would
// run rather than only counting them.
func explainAutomation(cfg, def *datamodel.Config) datamodel.ExplainSection {
	hooks := slices.Concat(cfg.Automation, cfg.UserAutomation)
	lines := make([]string, 0, len(hooks))
	for _, h := range hooks {
		state := ""
		if h.Enabled != nil && !*h.Enabled {
			state = " (disabled)"
		}
		lines = append(lines, string(h.On)+" -> "+h.Run+state)
	}
	if len(lines) == 0 {
		lines = []string{"none defined"}
	}
	differs := changed(cfg.Automation, def.Automation) || len(cfg.UserAutomation) > 0
	return datamodel.ExplainSection{Name: "automation", Provenance: tierOf(differs, datamodel.ProvenanceRepo), Lines: lines}
}

func orUnset(s string) string {
	if s == "" {
		return "(unset)"
	}
	return s
}

// changed reports whether a is not deeply equal to its default counterpart b.
func changed[T any](a, b T) bool { return !reflect.DeepEqual(a, b) }

// tierOf tags a section default when it matches the baked default, else with
// the loader-recorded tier that supplied it (repo when none was recorded,
// e.g. a config parsed without the user tier).
func tierOf(differs bool, tier datamodel.Provenance) datamodel.Provenance {
	if !differs {
		return datamodel.ProvenanceDefault
	}
	return cmp.Or(tier, datamodel.ProvenanceRepo)
}

func explainIdentity(cfg, def *datamodel.Config) datamodel.ExplainSection {
	boards := cfg.EffectiveBoards()
	lines := []string{
		fmt.Sprintf("project: %s (%s)", cfg.Project.Key, cfg.Project.Name),
		fmt.Sprintf("id style: %s", cfg.ID.Style),
	}
	for _, b := range boards {
		lines = append(lines, "board: "+describeBoard(b))
	}
	differs := changed(cfg.Project, def.Project) || changed(cfg.ID, def.ID) || changed(boards, def.EffectiveBoards())
	return datamodel.ExplainSection{Name: "identity", Provenance: tierOf(differs, datamodel.ProvenanceRepo), Lines: lines}
}

func describeBoard(b datamodel.Board) string {
	s := b.Key
	if b.Name != "" && b.Name != b.Key {
		s += " (" + b.Name + ")"
	}
	var flags []string
	if b.Default {
		flags = append(flags, "default")
	}
	if b.Archived {
		flags = append(flags, "archived")
	}
	if len(flags) > 0 {
		s += " [" + strings.Join(flags, ", ") + "]"
	}
	return s
}

func explainWorkflow(cfg, def *datamodel.Config, typ string) datamodel.ExplainSection {
	wf := cfg.Workflows[typ]
	lines := []string{
		"initial: " + wf.Initial,
		"wip_policy: " + string(wf.EffectiveWipPolicy()),
	}
	if wf.CloseTarget != "" {
		lines = append(lines, "close_target: "+wf.CloseTarget)
	}
	for _, st := range wf.States {
		lines = append(lines, describeState(wf, st))
	}
	defWF, isDefaultType := def.Workflows[typ]
	return datamodel.ExplainSection{
		Name:       "workflow: " + typ,
		Provenance: tierOf(!isDefaultType || changed(wf, defWF), datamodel.ProvenanceRepo),
		Lines:      lines,
	}
}

func describeState(wf datamodel.Workflow, st datamodel.State) string {
	line := fmt.Sprintf("%s (%s)", st.Key, st.Category)
	if st.Wip > 0 {
		line += fmt.Sprintf(", wip %d/%s", st.Wip, wf.EffectiveWipPolicy())
	}
	if st.Resolution != "" {
		line += ", resolution=" + st.Resolution
	}
	if st.Category == datamodel.CategoryDone {
		line += "; entering it requires blockers closed (a dangling or unknown blocker counts as satisfied and only warns)"
	}
	return line
}

func explainCommit(cfg, def *datamodel.Config) datamodel.ExplainSection {
	c := cfg.Commit
	lines := []string{
		"mode: " + string(c.Mode),
		fmt.Sprintf("subject: %q (up to %d ticket numbers listed, then a count)", commitSubjectTemplate(cfg, 1), subjectNumbersCap),
		fmt.Sprintf("trailer: %s (up to %d numbers)", c.Trailer, commitTrailerCap),
		"close_trailer: " + c.CloseTrailer,
	}
	if cfg.UserCommitSubject != "" {
		lines = append(lines, "subject template is a user override from ~/.config/kira/config.yaml")
	}
	for _, m := range datamodel.LinkMarkers {
		if slices.Contains(c.LinkMarkers, m) {
			lines = append(lines, "link marker: "+describeLinkMarker(m, c))
		}
	}
	for _, m := range c.ReferenceMarkers {
		lines = append(lines, "reference marker: "+describeReferenceMarker(m))
	}
	userSubject := cfg.UserCommitSubject != ""
	tier := datamodel.ProvenanceRepo
	if userSubject {
		tier = datamodel.ProvenanceUser
	}
	return datamodel.ExplainSection{Name: "commit", Provenance: tierOf(changed(c, def.Commit) || userSubject, tier), Lines: lines}
}

// describeLinkMarker documents a marker that associates a commit with a
// ticket (and thus can auto-close it). Precedence when multiple markers
// match is fixed — trailer, then subject, then leading_number — regardless
// of this list's order in config; the list only enables/disables markers.
func describeLinkMarker(m datamodel.LinkMarker, c datamodel.Commit) string {
	switch m {
	case datamodel.LinkMarkerTrailer:
		return fmt.Sprintf("trailer — a %q git trailer naming the ticket", c.Trailer)
	case datamodel.LinkMarkerSubject:
		return "subject — a `[KEY-n]` marker in the commit subject"
	case datamodel.LinkMarkerLeadingNumber:
		return "leading_number — a bare KEY-n token leading the subject"
	default:
		return string(m)
	}
}

// describeReferenceMarker documents a marker that only mentions a ticket
// (recorded for `kira log`/`kira show`, never auto-closes it).
func describeReferenceMarker(m datamodel.ReferenceMarker) string {
	switch m {
	case datamodel.ReferenceMarkerBare:
		return "bare — a KEY-n token anywhere in the commit body"
	default:
		return string(m)
	}
}

func explainVocab(cfg, def *datamodel.Config) datamodel.ExplainSection {
	lines := []string{
		fmt.Sprintf("system label: %s (auto-added by `kira create --here`; excluded from vocabulary checks)", datamodel.CapturedLabel),
		describeVocab("labels", cfg.Labels.Known, cfg.Labels.Strict),
		describeVocab("people", cfg.People.Names(), cfg.People.Strict),
		describeEnumVocab("priorities", cfg.Priorities, cfg.Labels.Strict),
		describeEnumVocab("subtypes", cfg.Subtypes, cfg.Labels.Strict),
		describeEnumVocab("resolutions", cfg.Resolutions, cfg.Labels.Strict),
	}
	if len(cfg.ResolutionsDropped) > 0 {
		lines = append(lines, "resolutions_dropped: "+strings.Join(cfg.ResolutionsDropped, ", ")+
			" (a done-category resolution treated as a non-completion outcome for epic progress)")
	}
	differs := vocabChanged(cfg.Labels, def.Labels) || peopleChanged(cfg.People, def.People) ||
		changed(cfg.Priorities, def.Priorities) || changed(cfg.Subtypes, def.Subtypes) ||
		changed(cfg.Resolutions, def.Resolutions) || changed(cfg.ResolutionsDropped, def.ResolutionsDropped)
	return datamodel.ExplainSection{Name: "vocab", Provenance: tierOf(differs, datamodel.ProvenanceRepo), Lines: lines}
}

// vocabChanged and peopleChanged treat a scaffolded `known: []` the same as
// an absent/nil list — reflect.DeepEqual would otherwise flag every freshly
// initialized repo as a "repo" override of an empty default vocabulary.
func vocabChanged(a, b datamodel.Vocab) bool {
	return a.Strict != b.Strict || !slices.Equal(a.Known, b.Known)
}

func peopleChanged(a, b datamodel.People) bool {
	return a.Strict != b.Strict || !slices.EqualFunc(a.Known, b.Known, personEqual)
}

func personEqual(a, b datamodel.Person) bool {
	return a.Name == b.Name && slices.Equal(a.Git, b.Git)
}

func describeVocab(field string, known []string, strict bool) string {
	mode := "advisory"
	if strict {
		mode = "strict"
	}
	values := "none"
	if len(known) > 0 {
		values = strings.Join(known, ", ")
	}
	return fmt.Sprintf("%s: %s (%s)", field, values, mode)
}

func describeEnumVocab(field string, v datamodel.EnumVocab, fallback bool) string {
	return describeVocab(field, v.Values, v.StrictOr(fallback))
}

func explainUI(cfg, def *datamodel.Config) datamodel.ExplainSection {
	lines := []string{
		fmt.Sprintf("icons: %s (override with the KIRA_ICONS env var: nerd|emoji|text|always|never)", cfg.UI.Icons),
		"color: " + string(cfg.UI.Color),
		"background: " + string(cfg.UI.Background),
	}
	return datamodel.ExplainSection{Name: "ui", Provenance: tierOf(changed(cfg.UI, def.UI), cfg.UIProvenance), Lines: lines}
}

func explainWorkon(cfg, def *datamodel.Config) datamodel.ExplainSection {
	lines := []string{
		"branch_pattern: " + cfg.Workon.BranchPattern,
		"casing: " + string(cfg.Workon.Casing),
		fmt.Sprintf("worktree: %t (dir %s)", cfg.Workon.Worktree, cfg.Workon.WorktreeDir),
	}
	return datamodel.ExplainSection{Name: "workon", Provenance: tierOf(changed(cfg.Workon, def.Workon), cfg.WorkonProvenance), Lines: lines}
}

func explainMerge(cfg, def *datamodel.Config) datamodel.ExplainSection {
	lines := []string{"policy: " + string(cfg.Merge.Policy)}
	return datamodel.ExplainSection{Name: "merge", Provenance: tierOf(changed(cfg.Merge, def.Merge), datamodel.ProvenanceRepo), Lines: lines}
}

func explainSync(cfg, def *datamodel.Config) datamodel.ExplainSection {
	lines := []string{
		fmt.Sprintf("push: %t", cfg.Sync.Push),
		"dirty: " + string(cfg.Sync.Dirty),
	}
	return datamodel.ExplainSection{Name: "sync", Provenance: tierOf(changed(cfg.Sync, def.Sync), datamodel.ProvenanceRepo), Lines: lines}
}
