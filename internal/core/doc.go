// Package core is kira's service layer: the single implementation of every
// mutation and read, called by cmd/kira and (later) the TUI/nvim alike, so
// semantics cannot drift between frontends (docs/design/01-architecture.md §6).
//
// A Store binds one repository's .kira/ tree and owns filesystem discovery,
// atomic ticket writes, the advisory lock, and git invocation. M0/M1 implement
// Init, Create, Show, Edit, List, Move, Assign, Link, Comment, Query/Tree,
// Find, and Discover; Log, Stats, Doctor, and Sync land in later milestones.
package core
