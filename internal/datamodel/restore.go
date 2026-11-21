package datamodel

type RestoreResult struct {
	ID       string        `json:"id"`
	Number   string        `json:"number"`
	Source   string        `json:"source"`
	DryRun   bool          `json:"dry_run"`
	Changes  []FieldChange `json:"changes"`
	Warnings []Warning     `json:"warnings,omitempty"`
}
