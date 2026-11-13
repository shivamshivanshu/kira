package config

import "github.com/shivamshivanshu/kira/internal/datamodel"

func Default() *datamodel.Config {
	return &datamodel.Config{
		Version: datamodel.InitialSchemaVersion,
		Project: datamodel.Project{Key: "KIRA", Name: "kira"},
		Labels:  datamodel.Vocab{},
		People:  datamodel.People{},
		Filters: map[string]string{},
		Sprints: nil,
		Commit:  datamodel.Commit{Mode: datamodel.CommitAuto},
		Merge:   datamodel.Merge{Policy: datamodel.MergeAuto},
		Sync:    datamodel.Sync{Push: false},
		Workon:  datamodel.Workon{BranchPattern: "{key}/{number}-{slug}", WorktreeDir: datamodel.DefaultWorktreeDir},
		UI: datamodel.UI{
			Icons:      datamodel.IconAuto,
			Background: datamodel.BackgroundDark,
			Color:      datamodel.ColorAuto,
			List:       datamodel.UIList{Columns: datamodel.DefaultListColumns},
			Tui:        datamodel.UITui{Split: datamodel.DefaultSplit, Refresh: datamodel.DefaultRefresh},
			AutoTUI:    true,
		},
		Git:      datamodel.Git{},
		Estimate: datamodel.Estimate{Unit: datamodel.EstimatePoints},
	}
}
