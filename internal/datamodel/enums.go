package datamodel

type IDStyle string

const (
	IDStyleSequential IDStyle = "sequential"
	IDStyleHash       IDStyle = "hash"
)

var IDStyles = []IDStyle{IDStyleSequential, IDStyleHash}

type CommitMode string

const (
	CommitAuto   CommitMode = "auto"
	CommitManual CommitMode = "manual"
	CommitPrompt CommitMode = "prompt"
)

var CommitModes = []CommitMode{CommitAuto, CommitManual, CommitPrompt}

type MergePolicy string

const (
	MergeAuto   MergePolicy = "auto"
	MergeManual MergePolicy = "manual"
)

var MergePolicies = []MergePolicy{MergeAuto, MergeManual}

type IconMode string

const (
	IconAuto   IconMode = "auto"
	IconAlways IconMode = "always"
	IconNever  IconMode = "never"
)

var IconModes = []IconMode{IconAuto, IconAlways, IconNever}

type EstimateUnit string

const (
	EstimatePoints EstimateUnit = "points"
	EstimateHours  EstimateUnit = "hours"
)

var EstimateUnits = []EstimateUnit{EstimatePoints, EstimateHours}

type Casing string

const (
	CasingKebab Casing = "kebab"
	CasingSnake Casing = "snake"
)

var Casings = []Casing{CasingKebab, CasingSnake}

type Category string

const (
	CategoryTodo  Category = "todo"
	CategoryDoing Category = "doing"
	CategoryDone  Category = "done"
)

var Categories = []Category{CategoryTodo, CategoryDoing, CategoryDone}
