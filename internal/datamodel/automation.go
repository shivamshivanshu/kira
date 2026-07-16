package datamodel

import (
	"fmt"
	"strings"
	"time"
)

const DefaultAutomationTimeout = 30 * time.Second

type EventName string

const (
	EventItemCreated      EventName = "item.created"
	EventItemStateChanged EventName = "item.state_changed"
	EventSyncCompleted    EventName = "sync.completed"
)

var AutomationEvents = []EventName{EventItemCreated, EventItemStateChanged, EventSyncCompleted}

type AutomationHook struct {
	Name    string           `yaml:"name" json:"name"`
	On      EventName        `yaml:"on" json:"on"`
	Run     string           `yaml:"run" json:"run"`
	Enabled *bool            `yaml:"enabled" json:"enabled"`
	Timeout string           `yaml:"timeout" json:"timeout"`
	Match   *AutomationMatch `yaml:"match" json:"match"`
}

type AutomationMatch struct {
	To   string `yaml:"to" json:"to"`
	From string `yaml:"from" json:"from"`
	Type string `yaml:"type" json:"type"`
}

func (h AutomationHook) IsEnabled() bool { return h.Enabled == nil || *h.Enabled }

func (h AutomationHook) TimeoutDuration() (time.Duration, error) {
	if strings.TrimSpace(h.Timeout) == "" {
		return DefaultAutomationTimeout, nil
	}
	d, err := time.ParseDuration(h.Timeout)
	if err != nil {
		return DefaultAutomationTimeout, err
	}
	if d <= 0 {
		return DefaultAutomationTimeout, fmt.Errorf("timeout must be positive, got %q", h.Timeout)
	}
	return d, nil
}

type AutomationSource string

const (
	AutomationSourceRepo AutomationSource = ""
	AutomationSourceUser AutomationSource = "user"
)

type AutomationHookView struct {
	Name    string           `json:"name"`
	On      EventName        `json:"on"`
	Run     string           `json:"run"`
	Enabled bool             `json:"enabled"`
	Source  AutomationSource `json:"source,omitempty"`
}

type AutomationListResult struct {
	Hooks   []AutomationHookView `json:"hooks"`
	Trusted bool                 `json:"trusted"`
}

type AutomationTrustResult struct {
	Hooks []AutomationHookView `json:"hooks"`
	Hash  string               `json:"hash"`
}
