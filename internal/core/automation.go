package core

import (
	"os"
	"slices"

	"github.com/shivamshivanshu/kira/internal/automation"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/syncx"
)

func (s *Store) AutomationList(cfg *datamodel.Config) *datamodel.AutomationListResult {
	views := automationViews(cfg.Automation, datamodel.AutomationSourceRepo)
	views = append(views, automationViews(cfg.UserAutomation, datamodel.AutomationSourceUser)...)
	return &datamodel.AutomationListResult{
		Hooks:   views,
		Trusted: automation.Trusted(s.fs().CacheDir(), cfg),
	}
}

func (s *Store) AutomationTrust(cfg *datamodel.Config) (*datamodel.AutomationTrustResult, error) {
	hash, err := automation.Grant(s.fs().CacheDir(), cfg)
	if err != nil {
		return nil, errx.User("recording automation trust: %v", err)
	}
	return &datamodel.AutomationTrustResult{Hooks: automationViews(cfg.Automation, datamodel.AutomationSourceRepo), Hash: hash}, nil
}

func automationViews(hooks []datamodel.AutomationHook, source datamodel.AutomationSource) []datamodel.AutomationHookView {
	views := make([]datamodel.AutomationHookView, len(hooks))
	for i, h := range hooks {
		views[i] = datamodel.AutomationHookView{Name: h.Name, On: h.On, Run: h.Run, Enabled: h.IsEnabled(), Source: source}
	}
	return views
}

func (s *Store) fire(cfg *datamodel.Config, ev automation.Event) {
	automation.Fire(os.Stderr, s.root, s.fs().CacheDir(), cfg, ev, s.actor)
}

func (s *Store) fireAutomation(cfg *datamodel.Config, cs *datamodel.ChangeSet, sha string) {
	ev, ok := s.eventFor(cfg, cs)
	if !ok {
		return
	}
	ev.Commit = sha
	s.fire(cfg, ev)
}

func (s *Store) fireSyncCompleted(cfg *datamodel.Config, report *syncx.Report) {
	if len(cfg.Automation) == 0 && len(cfg.UserAutomation) == 0 {
		return
	}
	s.fire(cfg, automation.Event{Name: datamodel.EventSyncCompleted, Source: datamodel.SourceSync, Sync: report})
}

func (s *Store) eventFor(cfg *datamodel.Config, cs *datamodel.ChangeSet) (automation.Event, bool) {
	switch cs.Kind {
	case datamodel.ChangeCreated:
		return s.baseEvent(cfg, datamodel.EventItemCreated, cs.Source, cs.After), true
	case datamodel.ChangeMutated:
		if !slices.Contains(cs.Changed, datamodel.KeyState) {
			return automation.Event{}, false
		}
		after := cs.After
		ev := s.baseEvent(cfg, datamodel.EventItemStateChanged, cs.Source, after)
		ev.Changes = changeMap(cs)
		ev.To = after.State
		ev.ToCategory = categoryString(cfg, after.Type, after.State)
		if cs.Before != nil {
			ev.From = cs.Before.State
			ev.FromCategory = categoryString(cfg, cs.Before.Type, cs.Before.State)
		}
		return ev, true
	}
	return automation.Event{}, false
}

func (s *Store) baseEvent(cfg *datamodel.Config, name datamodel.EventName, source datamodel.ChangeSource, it *datamodel.Item) automation.Event {
	snapshot := showResultOf(cfg, it)
	return automation.Event{
		Name:   name,
		Source: source,
		Item:   &snapshot,
		ItemID: it.ID,
		Number: it.Number,
		Type:   it.Type,
		Title:  it.Title,
	}
}

func changeMap(cs *datamodel.ChangeSet) map[string]automation.Change {
	if cs.Before == nil {
		return nil
	}
	m := make(map[string]automation.Change, len(cs.Changed))
	for _, f := range cs.Changed {
		if d, ok := datamodel.Field(f); ok {
			m[f] = automation.Change{Old: d.Get(cs.Before), New: d.Get(cs.After)}
		}
	}
	return m
}

func (s *Store) actor() automation.Actor {
	name, _ := s.repo().Output("config", "user.name")
	email, _ := s.repo().Output("config", "user.email")
	return automation.Actor{Name: name, Email: email}
}
