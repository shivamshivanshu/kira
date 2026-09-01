package datamodel

import "strings"

func sectionName(line string) (string, bool) { return strings.CutPrefix(line, "## ") }

type bodySpan struct{ start, end int }

// bodySectionSpans locates each "## <Heading>" span within lines: the heading
// line up to (not including) the next heading, or the end. It is the single
// source of the section-boundary rule; BodySections derives content from it.
func bodySectionSpans(lines []string) map[string]bodySpan {
	spans := make(map[string]bodySpan)
	name, start := "", -1
	closeSpan := func(end int) {
		if name != "" {
			spans[name] = bodySpan{start, end}
		}
	}
	for i, line := range lines {
		if title, ok := sectionName(line); ok {
			closeSpan(i)
			name, start = title, i
		}
	}
	closeSpan(len(lines))
	return spans
}

// BodySections maps each "## <Heading>" section to its trimmed content.
func BodySections(body string) map[string]string {
	lines := strings.Split(strings.TrimSuffix(body, "\n"), "\n")
	spans := bodySectionSpans(lines)
	out := make(map[string]string, len(spans))
	for name, span := range spans {
		out[name] = strings.TrimSpace(strings.Join(lines[span.start+1:span.end], "\n"))
	}
	return out
}

// ContainsSectionHeading reports whether s has a line BodySections would read
// as a "## " heading; such a line inside section content would split it.
func ContainsSectionHeading(s string) bool {
	for _, line := range strings.Split(s, "\n") {
		if _, ok := sectionName(line); ok {
			return true
		}
	}
	return false
}

// UpsertBodySection sets section's content within body, returning the new body.
// An existing section's content is replaced in place, leaving every other
// section byte-identical. A section not yet present is inserted immediately
// before a trailing "## Comments" section if one exists, else appended. Empty
// content removes the section.
func UpsertBodySection(body, section, content string) string {
	content = strings.TrimSpace(content)
	lines := strings.Split(strings.TrimSuffix(body, "\n"), "\n")
	spans := bodySectionSpans(lines)

	if span, ok := spans[section]; ok {
		if content == "" {
			return joinBody(splice(lines, removeStart(lines, span), span.end, nil))
		}
		block := renderBodySection(section, content, needsSeparator(lines, span.start), span.end == len(lines))
		return joinBody(splice(lines, span.start, span.end, block))
	}
	if content == "" {
		return body
	}
	at := len(lines)
	if comments, ok := spans["Comments"]; ok {
		at = comments.start
	}
	return joinBody(splice(lines, at, at, renderBodySection(section, content, needsSeparator(lines, at), at == len(lines))))
}

// removeStart extends a removed last section backward over its leading blank
// separator, so deleting it leaves no dangling trailing blank line.
func removeStart(lines []string, span bodySpan) int {
	if span.end == len(lines) && span.start > 0 && lines[span.start-1] == "" {
		return span.start - 1
	}
	return span.start
}

// needsSeparator reports whether a blank line must precede a "## " heading
// spliced in at lines[at]: true unless that point already follows a blank line.
func needsSeparator(lines []string, at int) bool {
	return at > 0 && lines[at-1] != ""
}

func renderBodySection(name, content string, leadingSeparator, isLast bool) []string {
	block := append([]string{"## " + name, ""}, strings.Split(content, "\n")...)
	if !isLast {
		block = append(block, "")
	}
	if leadingSeparator {
		block = append([]string{""}, block...)
	}
	return block
}

func splice(lines []string, start, end int, block []string) []string {
	out := make([]string, 0, len(lines)-(end-start)+len(block))
	out = append(out, lines[:start]...)
	out = append(out, block...)
	out = append(out, lines[end:]...)
	return out
}

func joinBody(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, "\n") + "\n"
}
