package config

import "github.com/shivamshivanshu/kira/internal/datamodel"

func Default() *datamodel.Config {
	return &datamodel.Config{
		Version: datamodel.InitialSchemaVersion,
		Project: datamodel.Project{Key: "KIRA", Name: "kira"},
		ID:      datamodel.Identity{Style: datamodel.IDStyleSequential},
		Workflows: map[string]datamodel.Workflow{
			"ticket": {
				States: []datamodel.State{
					{Key: "TODO", Category: datamodel.CategoryTodo},
					{Key: "IN_PROGRESS", Category: datamodel.CategoryDoing, Wip: 3},
					{Key: "REVIEW", Category: datamodel.CategoryDoing, Wip: 2},
					{Key: "DONE", Category: datamodel.CategoryDone},
					{Key: "WONT_DO", Category: datamodel.CategoryDone, Resolution: datamodel.ResolutionDropped},
				},
				Initial: "TODO",
				Transitions: map[string][]datamodel.Transition{
					"TODO":        datamodel.TransitionsTo("IN_PROGRESS", "WONT_DO"),
					"IN_PROGRESS": datamodel.TransitionsTo("REVIEW", "TODO", "WONT_DO"),
					"REVIEW": {
						{To: "DONE", Require: []string{"resolution"}, Set: map[string]string{"resolution": "done"}},
						{To: "IN_PROGRESS"},
					},
					"DONE":    {},
					"WONT_DO": {},
				},
				EnforceTransitions: true,
			},
			"epic": {
				States: []datamodel.State{
					{Key: "PLANNED", Category: datamodel.CategoryTodo},
					{Key: "ACTIVE", Category: datamodel.CategoryDoing},
					{Key: "DONE", Category: datamodel.CategoryDone},
				},
				Initial: "PLANNED",
				Transitions: map[string][]datamodel.Transition{
					"PLANNED": datamodel.TransitionsTo("ACTIVE"),
					"ACTIVE":  datamodel.TransitionsTo("DONE"),
					"DONE":    {},
				},
			},
		},
		Labels:      datamodel.Vocab{},
		People:      datamodel.People{},
		Priorities:  datamodel.EnumVocab{Values: []string{"P0", "P1", "P2", "P3"}},
		Subtypes:    datamodel.EnumVocab{Values: []string{"bug", "story", "task", "spike"}},
		Resolutions: datamodel.EnumVocab{Values: []string{"done", datamodel.ResolutionDropped, "duplicate", "cannot-reproduce"}},

		ResolutionsDropped: []string{datamodel.ResolutionDropped},
		Filters:            map[string]string{},
		Sprints:            nil,
		Commit: datamodel.Commit{
			Mode:             datamodel.CommitAuto,
			Trailer:          "Kira-Ticket",
			CloseTrailer:     "Kira-Closes",
			SubjectPrefix:    "kira: ",
			LinkMarkers:      []datamodel.LinkMarker{datamodel.LinkMarkerTrailer, datamodel.LinkMarkerSubject, datamodel.LinkMarkerLeadingNumber},
			ReferenceMarkers: []datamodel.ReferenceMarker{datamodel.ReferenceMarkerBare},
		},
		Merge:  datamodel.Merge{Policy: datamodel.MergeAuto},
		Sync:   datamodel.Sync{Push: false, Dirty: datamodel.SyncDirtyAuto},
		Workon: datamodel.Workon{BranchPattern: "{key}/{number}-{slug}", Casing: datamodel.CasingKebab, WorktreeDir: datamodel.DefaultWorktreeDir},
		UI: datamodel.UI{
			Icons:      datamodel.IconAuto,
			Background: datamodel.BackgroundAuto,
			Color:      datamodel.ColorAuto,
			List:       datamodel.UIList{Columns: datamodel.DefaultListColumns},
			Tui:        datamodel.UITui{Split: datamodel.DefaultSplit, Refresh: datamodel.DefaultRefresh},
			AutoTUI:    true,
		},
		Git:      datamodel.Git{},
		Estimate: datamodel.Estimate{Unit: datamodel.EstimatePoints},
		Fields:   map[string]any{},
	}
}
