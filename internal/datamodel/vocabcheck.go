package datamodel

import "slices"

type VocabViolation struct {
	Field  string
	Vocab  string
	Value  string
	Strict bool
}

// VocabViolations reports values outside their vocabulary in the order the
// fields were historically validated (subtype, priority, resolution, owner,
// reporter, labels), which is the order mutation errors print in.
func VocabViolations(cfg *Config, it *Item) []VocabViolation {
	var out []VocabViolation
	checkScalar := func(field, vocab string, value *string, known []string, strict bool) {
		if value != nil && *value != "" && !slices.Contains(known, *value) {
			out = append(out, VocabViolation{Field: field, Vocab: vocab, Value: *value, Strict: strict})
		}
	}
	checkScalar(KeySubtype, "subtype", it.Subtype, Subtypes, cfg.Labels.Strict)
	checkScalar(KeyPriority, "priority", it.Priority, Priorities, cfg.Labels.Strict)
	checkScalar(KeyResolution, "resolution", it.Resolution, Resolutions, cfg.Labels.Strict)
	people := cfg.People.Names()
	checkScalar(KeyOwner, "person", it.Owner, people, cfg.People.Strict)
	checkScalar(KeyReporter, "person", it.Reporter, people, cfg.People.Strict)
	for _, label := range it.Labels {
		if label != CapturedLabel && !slices.Contains(cfg.Labels.Known, label) {
			out = append(out, VocabViolation{Field: KeyLabels, Vocab: "label", Value: label, Strict: cfg.Labels.Strict})
		}
	}
	return out
}
