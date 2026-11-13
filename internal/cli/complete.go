package cli

import (
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/shivamshivanshu/kira/internal/config"
	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/setx"
	"github.com/shivamshivanshu/kira/internal/showfmt"
)

const completionCap = 200

type completer = func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective)

func attachCompletions(root *cobra.Command, g *globalFlags) {
	items := completeItems(g, anyItem)
	containers := completeItems(g, containerItemsOnly)
	people := completeVocab(g, func(c *datamodel.Config) []string { return c.People.Names() })
	labels := completeVocab(g, func(c *datamodel.Config) []string { return c.Labels.Known })
	subtypes := completeStatic(datamodel.Subtypes)
	sprints := completeVocab(g, sprintKeys)
	activeBoards := completeVocab(g, activeBoardKeys)
	allBoards := completeVocab(g, func(c *datamodel.Config) []string { return c.BoardKeys() })

	for _, name := range []string{"show", "edit", "comment", "log", "blame", "workon", "link"} {
		if c := subCmd(root, name); c != nil {
			c.ValidArgsFunction = positional(items)
		}
	}
	if c := subCmd(root, "create"); c != nil {
		c.ValidArgsFunction = positional(completeStatic(datamodel.ItemTypes))
	}
	if c := subCmd(root, "move"); c != nil {
		c.ValidArgsFunction = positional(items, completeMoveTarget(g))
	}
	if c := subCmd(root, "assign"); c != nil {
		c.ValidArgsFunction = positional(items, people)
	}
	if c := subCmd(root, "tree"); c != nil {
		c.ValidArgsFunction = positional(items)
	}
	if c := subCmd(root, "stats"); c != nil {
		c.ValidArgsFunction = positional(containers)
	}
	if c := subCmd(subCmd(root, "config"), "set"); c != nil {
		c.ValidArgsFunction = positional(completeStatic(config.SetKeys()))
	}
	if lbl := subCmd(root, "label"); lbl != nil {
		for _, name := range []string{"add", "rm"} {
			if c := subCmd(lbl, name); c != nil {
				c.ValidArgsFunction = positional(items, labels)
			}
		}
	}
	if sp := subCmd(root, "sprint"); sp != nil {
		setArgs(sp, "activate", positional(sprints))
		setArgs(sp, "close", positional(sprints))
	}
	if b := subCmd(root, "board"); b != nil {
		b.ValidArgsFunction = positional(containers)
		setArgs(b, "rename", positional(allBoards))
		setArgs(b, "archive", positional(activeBoards))
		setArgs(b, "unarchive", positional(allBoards))
		setArgs(b, "move", positional(items, activeBoards))
	}

	flags := map[string]completer{
		"label":      labels,
		"priority":   completeStatic(datamodel.Priorities),
		"owner":      people,
		"reporter":   people,
		"sprint":     sprints,
		"filter":     completeVocab(g, filterNames),
		"resolution": completeStatic(datamodel.Resolutions),
		"subtype":    subtypes,
		"state":      completeStatic(workflowStates()),
		"format":     completeStatic(showfmt.Names()),
		"category":   completeStatic(categoryValues()),
		"type":       completeStatic(datamodel.ItemTypes),
		"parent":     containers,
		"blocked-by": items,
		"board":      activeBoards,
	}
	walk(root, func(c *cobra.Command) {
		for name, fn := range flags {
			if _, bound := c.GetFlagCompletionFunc(name); bound || c.Flags().Lookup(name) == nil {
				continue
			}
			_ = c.RegisterFlagCompletionFunc(name, fn)
		}
		for _, lt := range datamodel.LinkTypes {
			if fl := core.FlagForLinkType(string(lt)); c.Flags().Lookup(fl) != nil {
				_ = c.RegisterFlagCompletionFunc(fl, items)
			}
		}
	})
}

func setArgs(parent *cobra.Command, name string, fn completer) {
	if c := subCmd(parent, name); c != nil {
		c.ValidArgsFunction = fn
	}
}

type itemScope int

const (
	anyItem itemScope = iota
	containerItemsOnly
)

func completeItems(g *globalFlags, scope itemScope) completer {
	return func(_ *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		items, ok := cachedItems(g)
		if !ok {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		pref := strings.ToLower(toComplete)
		out := make([]string, 0, completionCap)
		for _, it := range items {
			if scope == containerItemsOnly && !datamodel.IsContainer(it.Type) {
				continue
			}
			if !strings.HasPrefix(strings.ToLower(it.Number), pref) {
				continue
			}
			out = append(out, it.Number+"\t"+it.Title)
			if len(out) == completionCap {
				break
			}
		}
		return out, cobra.ShellCompDirectiveNoFileComp
	}
}

func completeMoveTarget(g *globalFlags) completer {
	return func(_ *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		items, ok := cachedItems(g)
		if !ok {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		it := findCached(items, args[0])
		if it == nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return prefixed(core.MoveTargets(it.Type), toComplete), cobra.ShellCompDirectiveNoFileComp
	}
}

func completeVocab(g *globalFlags, pick func(*datamodel.Config) []string) completer {
	return func(_ *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		_, cfg, err := openStore(g)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return prefixed(pick(cfg), toComplete), cobra.ShellCompDirectiveNoFileComp
	}
}

func completeStatic(values []string) completer {
	return func(_ *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return prefixed(values, toComplete), cobra.ShellCompDirectiveNoFileComp
	}
}

func cachedItems(g *globalFlags) ([]*datamodel.Item, bool) {
	s, _, err := openStore(g)
	if err != nil {
		return nil, false
	}
	items, err := s.CachedItems()
	if err != nil {
		return nil, false
	}
	return items, true
}

func findCached(items []*datamodel.Item, ref string) *datamodel.Item {
	for _, it := range items {
		if core.RefMatches(it, ref) {
			return it
		}
	}
	return nil
}

func prefixed(values []string, toComplete string) []string {
	out := make([]string, 0, len(values))
	for _, v := range values {
		if strings.HasPrefix(v, toComplete) {
			out = append(out, v)
			if len(out) == completionCap {
				break
			}
		}
	}
	return out
}

func positional(fns ...completer) completer {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) >= len(fns) {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return fns[len(args)](cmd, args, toComplete)
	}
}

func activeBoardKeys(cfg *datamodel.Config) []string {
	boards := cfg.ActiveBoards()
	keys := make([]string, len(boards))
	for i, b := range boards {
		keys[i] = b.Key
	}
	return keys
}

func sprintKeys(cfg *datamodel.Config) []string {
	out := []string{"active"}
	for _, sp := range cfg.Sprints {
		out = append(out, sp.Key)
	}
	return out
}

func filterNames(cfg *datamodel.Config) []string {
	out := make([]string, 0, len(cfg.Filters))
	for k := range cfg.Filters {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}

func workflowStates() []string {
	dedup := setx.NewDeduper[string]()
	var out []string
	for _, typ := range datamodel.ItemTypes {
		wf, _ := datamodel.WorkflowOf(typ)
		for _, st := range wf.States {
			if dedup.Add(st.Key) {
				out = append(out, st.Key)
			}
		}
	}
	slices.Sort(out)
	return out
}

func categoryValues() []string {
	out := make([]string, len(datamodel.Categories))
	for i, c := range datamodel.Categories {
		out[i] = string(c)
	}
	return out
}

func subCmd(parent *cobra.Command, name string) *cobra.Command {
	if parent == nil {
		return nil
	}
	for _, c := range parent.Commands() {
		if c.Name() == name {
			return c
		}
	}
	return nil
}

func walk(cmd *cobra.Command, fn func(*cobra.Command)) {
	fn(cmd)
	for _, c := range cmd.Commands() {
		walk(c, fn)
	}
}
