package gitx

import "strings"

type Commit struct {
	SHA          string
	Subject      string
	Author       string
	Timestamp    string
	Tickets      []string
	Closes       []string
	TrailerBlock string
	Body         string
}

const (
	nulFmt   = "%x00"
	nul      = "\x00"
	valueFmt = "%x1f"
	valueSep = "\x1f"
	nFields  = 8
)

func (r Repo) LogTrailers(rangeExpr, ticketKey, closeKey string) ([]Commit, error) {
	format := nulFmt + "%H" +
		nulFmt + "%s" +
		nulFmt + "%an" +
		nulFmt + "%cI" +
		nulFmt + "%(trailers:key=" + ticketKey + ",valueonly,separator=" + valueFmt + ")" +
		nulFmt + "%(trailers:key=" + closeKey + ",valueonly,separator=" + valueFmt + ")" +
		nulFmt + "%(trailers:only=true)" +
		nulFmt + "%B"
	out, err := r.OutputRaw("log", rangeExpr, "--pretty=format:"+format)
	if err != nil {
		return nil, err
	}
	tokens := strings.Split(out, nul)
	var commits []Commit
	for base := 1; base+nFields <= len(tokens); base += nFields {
		f := tokens[base : base+nFields]
		commits = append(commits, Commit{
			SHA:          f[0],
			Subject:      f[1],
			Author:       f[2],
			Timestamp:    f[3],
			Tickets:      splitValues(f[4]),
			Closes:       splitValues(f[5]),
			TrailerBlock: f[6],
			Body:         f[7],
		})
	}
	return commits, nil
}

func splitValues(field string) []string {
	var out []string
	for _, v := range strings.Split(field, valueSep) {
		if v = strings.TrimSpace(v); v != "" {
			out = append(out, v)
		}
	}
	return out
}
