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
	GitInstalled          bool
	InsideWorkTree        bool
	TrackedHooks          []string
	InstalledHooks        []string
	DriftedHooks          []string
	MergeDriverRegistered bool
	TicketAttrRegistered  bool
	MissingOptionalBins   []string
	Freshness             *Freshness
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
		return append(out, info(ClassEnv, "not inside a git work tree; skipping hook, merge-driver, and index-freshness checks"))
	}
	out = append(out, hookFindings(env)...)
	out = append(out, freshnessFinding(env.Freshness))
	return out
}

func hookFindings(env Env) []Finding {
	if len(env.TrackedHooks) == 0 {
		return []Finding{info(ClassHooks, "no tracked hooks in .kira/hooks")}
	}
	installed := hookSet(env.InstalledHooks)
	drifted := hookSet(env.DriftedHooks)
	var out []Finding
	for _, h := range env.TrackedHooks {
		switch {
		case installed[h]:
		case drifted[h]:
			out = append(out, warn(ClassHooks, "tracked hook "+h+" already runs kira alongside other commands; remove kira's lines and re-run `kira hooks install`, or keep managing it by hand"))
		default:
			out = append(out, info(ClassHooks, "tracked hook "+h+" is not installed in .git/hooks; run `kira hooks install`"))
		}
	}
	if !env.MergeDriverRegistered {
		out = append(out, info(ClassHooks, "kira merge driver is not registered in .git/config; run `kira hooks install`"))
	}
	if !env.TicketAttrRegistered {
		out = append(out, info(ClassHooks, "ticket merge attribute is not registered in .git/info/attributes; run `kira hooks install`"))
	}
	return out
}

func hookSet(names []string) map[string]bool {
	set := make(map[string]bool, len(names))
	for _, n := range names {
		set[n] = true
	}
	return set
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
