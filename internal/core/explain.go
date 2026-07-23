package core

import (
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
	)
	return &datamodel.ExplainResult{Sections: sections}
}

// changed reports whether a is not deeply equal to its default counterpart b.
func changed[T any](a, b T) bool { return !reflect.DeepEqual(a, b) }

// tierOf tags a section default when it matches the baked default, user when
// it differs and belongs to a user-config-tier key (ui, workon), else repo.
func tierOf(differs, userTier bool) datamodel.Provenance {
	switch {
	case !differs:
		return datamodel.ProvenanceDefault
	case userTier:
		return datamodel.ProvenanceUser
	default:
		return datamodel.ProvenanceRepo
	}
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
	return datamodel.ExplainSection{Name: "identity", Provenance: tierOf(differs, false), Lines: lines}
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
		fmt.Sprintf("enforce_transitions: %t", wf.EnforceTransitions),
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
		Provenance: tierOf(!isDefaultType || changed(wf, defWF), false),
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
	line += ": " + transitionDescription(wf, st.Key)
	if requiresBlockersClosed(wf, st.Key) {
		line += "; blockers_closed: a dangling or unknown blocker counts as satisfied (warns, doesn't block)"
	}
	return line
}

func transitionDescription(wf datamodel.Workflow, from string) string {
	if wf.EnforceTransitions {
		return transitionHint(wf, from)
	}
	targets := allowedTargets(wf, from)
	if len(targets) == 0 {
		return "any state (transitions not enforced)"
	}
	return "suggested " + strings.Join(targets, ", ") + "; any state allowed (transitions not enforced)"
}

func requiresBlockersClosed(wf datamodel.Workflow, from string) bool {
	for _, t := range wf.Transitions[from] {
		if slices.Contains(t.Require, datamodel.RequireBlockersClosed) {
			return true
		}
	}
	return false
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
	return datamodel.ExplainSection{Name: "commit", Provenance: tierOf(changed(c, def.Commit) || userSubject, userSubject), Lines: lines}
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
	return datamodel.ExplainSection{Name: "vocab", Provenance: tierOf(differs, false), Lines: lines}
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
	return datamodel.ExplainSection{Name: "ui", Provenance: tierOf(changed(cfg.UI, def.UI), true), Lines: lines}
}

func explainWorkon(cfg, def *datamodel.Config) datamodel.ExplainSection {
	lines := []string{
		"branch_pattern: " + cfg.Workon.BranchPattern,
		"casing: " + string(cfg.Workon.Casing),
		fmt.Sprintf("worktree: %t (dir %s)", cfg.Workon.Worktree, cfg.Workon.WorktreeDir),
	}
	return datamodel.ExplainSection{Name: "workon", Provenance: tierOf(changed(cfg.Workon, def.Workon), true), Lines: lines}
}

func explainMerge(cfg, def *datamodel.Config) datamodel.ExplainSection {
	lines := []string{"policy: " + string(cfg.Merge.Policy)}
	return datamodel.ExplainSection{Name: "merge", Provenance: tierOf(changed(cfg.Merge, def.Merge), false), Lines: lines}
}

func explainSync(cfg, def *datamodel.Config) datamodel.ExplainSection {
	lines := []string{
		fmt.Sprintf("push: %t", cfg.Sync.Push),
		"dirty: " + string(cfg.Sync.Dirty),
	}
	return datamodel.ExplainSection{Name: "sync", Provenance: tierOf(changed(cfg.Sync, def.Sync), false), Lines: lines}
}
