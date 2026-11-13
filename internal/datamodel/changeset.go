package datamodel

type ChangeSet struct {
	After   *Item
	Paths   []string
	Subject string
}
