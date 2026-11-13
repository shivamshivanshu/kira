package datamodel

type CommitMode string

const (
	CommitAuto   CommitMode = "auto"
	CommitManual CommitMode = "manual"
)

var CommitModes = []CommitMode{CommitAuto, CommitManual}

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
	IconAuto  IconMode = "auto"
	IconNerd  IconMode = "nerd"
	IconEmoji IconMode = "emoji"
	IconText  IconMode = "text"
)

var IconModes = []IconMode{IconAuto, IconNerd, IconEmoji, IconText}

type Background string

const (
	BackgroundDark  Background = "dark"
	BackgroundLight Background = "light"
)

var Backgrounds = []Background{BackgroundDark, BackgroundLight}

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

type SyncDirty string

const (
	SyncDirtyAuto   SyncDirty = "auto"
	SyncDirtyFail   SyncDirty = "fail"
	SyncDirtyCommit SyncDirty = "commit"
	SyncDirtyStash  SyncDirty = "stash"
)

var SyncDirties = []SyncDirty{SyncDirtyAuto, SyncDirtyFail, SyncDirtyCommit, SyncDirtyStash}

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

var LogKinds = []LogKind{LogKindEvent, LogKindCommit}

type BlameSource string

const (
	BlameSourceCommit    BlameSource = "commit"
	BlameSourceCreated   BlameSource = "created"
	BlameSourceSynthetic BlameSource = "synthetic"
)

var BlameSources = []BlameSource{BlameSourceCommit, BlameSourceCreated, BlameSourceSynthetic}
