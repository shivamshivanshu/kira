package datamodel

type ChangeKind string

const (
	ChangeCreated   ChangeKind = "created"
	ChangeMutated   ChangeKind = "mutated"
	ChangeCommented ChangeKind = "commented"
)

type ChangeSource string

const (
	SourceCLI     ChangeSource = "cli"
	SourceTrailer ChangeSource = "trailer"
	SourceSync    ChangeSource = "sync"
)

type ChangeSet struct {
	Kind    ChangeKind
	Before  *Item
	After   *Item
	Changed []string
	Paths   []string
	Subject string
	Source  ChangeSource
}
