package config

import "github.com/shivamshivanshu/kira/internal/datamodel"

func Default() *datamodel.Config {
	return &datamodel.Config{
		Version: datamodel.InitialSchemaVersion,
		Project: datamodel.Project{Key: "KIRA", Name: "kira"},
		ID:      datamodel.Identity{Style: datamodel.IDStyleSequential},
		Labels:  datamodel.Vocab{},
		People:  datamodel.People{},
		Filters: map[string]string{},
		Sprints: nil,
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
	}
}
