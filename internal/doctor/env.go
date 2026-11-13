package doctor

type Freshness struct {
	Built  bool   `json:"built"`
	Fresh  bool   `json:"fresh"`
	Reason string `json:"reason"`
}

type FreshnessReporter interface {
	Freshness() (Freshness, error)
}

func ResolveFreshness(r FreshnessReporter) *Freshness {
	if r == nil {
		return nil
	}
	f, err := r.Freshness()
	if err != nil {
		return nil
	}
	return &f
}

type Env struct {
	GitInstalled        bool
	InsideWorkTree      bool
	MissingOptionalBins []string
	Freshness           *Freshness
	// LegacySchemaDirPresent reports a leftover .kira/schema directory from
	// when item types were declared there.
	LegacySchemaDirPresent bool
}

func info(class Class, msg string) Finding {
	return Finding{Class: class, Severity: SeverityInfo, Message: msg}
}

func warn(class Class, msg string) Finding {
	return Finding{Class: class, Severity: SeverityWarning, Message: msg}
}

func envFindings(env Env) []Finding {
	var out []Finding
	if !env.GitInstalled {
		return []Finding{{
			Class:    ClassEnv,
			Severity: SeverityError,
			Message:  "git is not installed or not on PATH; kira needs git for commits, history, and index freshness",
		}}
	}
	for _, bin := range env.MissingOptionalBins {
		out = append(out, info(ClassEnv, bin+" not found on PATH; the feature that uses it falls back to a slower pure-Go path"))
	}
	if !env.InsideWorkTree {
		return append(out, info(ClassEnv, "not inside a git work tree; skipping the index-freshness check"))
	}
	out = append(out, freshnessFinding(env.Freshness))
	return out
}

func freshnessFinding(f *Freshness) Finding {
	switch {
	case f == nil:
		return info(ClassFreshness, "index freshness not checked (index unavailable)")
	case !f.Built:
		return info(ClassFreshness, "index not built yet; it builds on first read")
	case f.Fresh:
		return info(ClassFreshness, "index is up to date")
	default:
		return warn(ClassFreshness, "index is stale ("+f.Reason+"); run `kira index`")
	}
}
