package tui

const jumplistCap = 50

type jumpEntry struct {
	view   view
	itemID string
}

type jumplist struct {
	entries []jumpEntry
	pos     int
}

func (j *jumplist) push(e jumpEntry) {
	if n := len(j.entries); n > 0 && j.pos < n {
		j.entries = j.entries[:j.pos]
	}
	if n := len(j.entries); n > 0 && j.entries[n-1] == e {
		return
	}
	j.entries = append(j.entries, e)
	if len(j.entries) > jumplistCap {
		j.entries = j.entries[len(j.entries)-jumplistCap:]
	}
	j.pos = len(j.entries)
}

func (j *jumplist) back() (jumpEntry, bool) {
	if j.pos <= 0 || len(j.entries) == 0 {
		return jumpEntry{}, false
	}
	j.pos--
	return j.entries[j.pos], true
}

func (j *jumplist) forward() (jumpEntry, bool) {
	if j.pos >= len(j.entries)-1 {
		return jumpEntry{}, false
	}
	j.pos++
	return j.entries[j.pos], true
}

func (j *jumplist) dropMissing(exists func(string) bool) {
	kept := j.entries[:0]
	for _, e := range j.entries {
		if e.itemID == "" || exists(e.itemID) {
			kept = append(kept, e)
		}
	}
	j.entries = kept
	j.pos = clamp(j.pos, 0, len(j.entries))
}
