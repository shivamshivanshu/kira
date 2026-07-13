package automation

import "github.com/shivamshivanshu/kira/internal/datamodel"

func Matches(hook datamodel.AutomationHook, ev Event) bool {
	if hook.On != ev.Name {
		return false
	}
	m := hook.Match
	if m == nil {
		return true
	}
	if m.Type != "" && m.Type != ev.Type {
		return false
	}
	if m.To != "" && m.To != ev.To && m.To != ev.ToCategory {
		return false
	}
	if m.From != "" && m.From != ev.From && m.From != ev.FromCategory {
		return false
	}
	return true
}
