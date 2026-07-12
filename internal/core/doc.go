// Package core is the service layer: one function per verb (Create, Show, Edit,
// Move, Assign, Link, Comment, List, Query, Find, Log, Stats, Doctor, Sync),
// called by cmd/kira and the TUI alike so mutation semantics cannot drift
// between frontends. Verbs land in later work packages.
package core
