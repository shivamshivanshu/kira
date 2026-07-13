package datamodel

import (
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
	Name    string           `yaml:"name"`
	On      EventName        `yaml:"on"`
	Run     string           `yaml:"run"`
	Enabled *bool            `yaml:"enabled"`
	Timeout string           `yaml:"timeout"`
	Match   *AutomationMatch `yaml:"match"`
}

type AutomationMatch struct {
	To   string `yaml:"to"`
	From string `yaml:"from"`
	Type string `yaml:"type"`
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
	return d, nil
}

type AutomationHookView struct {
	Name    string    `json:"name"`
	On      EventName `json:"on"`
	Run     string    `json:"run"`
	Enabled bool      `json:"enabled"`
}

type AutomationListResult struct {
	Hooks   []AutomationHookView `json:"hooks"`
	Trusted bool                 `json:"trusted"`
}

type AutomationTrustResult struct {
	Hooks []AutomationHookView `json:"hooks"`
	Hash  string               `json:"hash"`
}
