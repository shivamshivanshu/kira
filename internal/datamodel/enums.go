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

type LinkMarker string

const (
	LinkMarkerTrailer       LinkMarker = "trailer"
	LinkMarkerSubject       LinkMarker = "subject"
	LinkMarkerLeadingNumber LinkMarker = "leading_number"
)

var LinkMarkers = []LinkMarker{LinkMarkerTrailer, LinkMarkerSubject, LinkMarkerLeadingNumber}

type ReferenceMarker string

const ReferenceMarkerBare ReferenceMarker = "bare"

var ReferenceMarkers = []ReferenceMarker{ReferenceMarkerBare}

type MergePolicy string

const (
	MergeAuto   MergePolicy = "auto"
	MergeManual MergePolicy = "manual"
)

var MergePolicies = []MergePolicy{MergeAuto, MergeManual}

type IconMode string

const (
	IconAuto   IconMode = "auto"
	IconNerd   IconMode = "nerd"
	IconEmoji  IconMode = "emoji"
	IconText   IconMode = "text"
	IconAlways IconMode = "always"
	IconNever  IconMode = "never"
)

var IconModes = []IconMode{IconAuto, IconNerd, IconEmoji, IconText, IconAlways, IconNever}

type Background string

const (
	BackgroundAuto  Background = "auto"
	BackgroundDark  Background = "dark"
	BackgroundLight Background = "light"
)

var Backgrounds = []Background{BackgroundAuto, BackgroundDark, BackgroundLight}

type ColorMode string

const (
	ColorAuto   ColorMode = "auto"
	ColorAlways ColorMode = "always"
	ColorNever  ColorMode = "never"
)

var ColorModes = []ColorMode{ColorAuto, ColorAlways, ColorNever}

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

func (c Casing) Separator() string {
	if c == CasingSnake {
		return "_"
	}
	return "-"
}

type SyncDirty string

const (
	SyncDirtyAuto   SyncDirty = "auto"
	SyncDirtyFail   SyncDirty = "fail"
	SyncDirtyCommit SyncDirty = "commit"
	SyncDirtyStash  SyncDirty = "stash"
)

var SyncDirties = []SyncDirty{SyncDirtyAuto, SyncDirtyFail, SyncDirtyCommit, SyncDirtyStash}

type WipPolicy string

const (
	WipWarn  WipPolicy = "warn"
	WipBlock WipPolicy = "block"
)

var WipPolicies = []WipPolicy{WipWarn, WipBlock}

type Category string

const (
	CategoryTodo  Category = "todo"
	CategoryDoing Category = "doing"
	CategoryDone  Category = "done"
)

var Categories = []Category{CategoryTodo, CategoryDoing, CategoryDone}

type LogKind string

const (
	LogKindEvent  LogKind = "event"
	LogKindCommit LogKind = "commit"
)

type BlameSource string

const (
	BlameSourceCommit    BlameSource = "commit"
	BlameSourceCreated   BlameSource = "created"
	BlameSourceSynthetic BlameSource = "synthetic"
)
