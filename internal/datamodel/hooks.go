package datamodel

type HookStatus struct {
	Name      string `json:"name"`
	Installed bool   `json:"installed"`
	Chained   bool   `json:"chained"`
}

type HooksInstallResult struct {
	Hooks       []HookStatus `json:"hooks"`
	MergeDriver bool         `json:"merge_driver"`
}

type HooksValidateResult struct {
	Hooks       []HookStatus `json:"hooks"`
	MergeDriver bool         `json:"merge_driver"`
	OK          bool         `json:"ok"`
}
