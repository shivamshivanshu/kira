package merge

type Side int

const (
	Ours Side = iota
	Theirs
)

type TextMerger func(base, ours, theirs string) (merged string, conflict bool)
