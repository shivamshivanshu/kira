package cli

import (
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/shivamshivanshu/kira/internal/config"
	"github.com/shivamshivanshu/kira/internal/core"
	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/showfmt"
)

const completionCap = 200

type completer = func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective)

func attachCompletions(root *cobra.Command, g *globalFlags) {
	items := completeItems(g, "")
	people := completeVocab(g, func(c *datamodel.Config) []string { return c.People.Names() })
	labels := completeVocab(g, func(c *datamodel.Config) []string { return c.Labels.Known })

	for _, name := range []string{"show", "edit", "comment", "log", "blame", "workon", "link"} {
		if c := subCmd(root, name); c != nil {
			c.ValidArgsFunction = positional(items)
		}
	}
	if c := subCmd(root, "create"); c != nil {
		c.ValidArgsFunction = positional(completeVocab(g, workflowTypes))
	}
	if c := subCmd(root, "move"); c != nil {
		c.ValidArgsFunction = positional(items, completeMoveTarget(g))
	}
	if c := subCmd(root, "assign"); c != nil {
		c.ValidArgsFunction = positional(items, people)
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

	flags := map[string]completer{
		"label":      labels,
		"priority":   completeVocab(g, func(c *datamodel.Config) []string { return c.Priorities.Values }),
		"owner":      people,
		"reporter":   people,
		"sprint":     completeVocab(g, sprintKeys),
		"filter":     completeVocab(g, filterNames),
		"resolution": completeVocab(g, func(c *datamodel.Config) []string { return c.Resolutions.Values }),
		"subtype":    completeVocab(g, func(c *datamodel.Config) []string { return c.Subtypes.Values }),
		"state":      completeVocab(g, workflowStates),
		"format":     completeStatic(showfmt.Names()),
		"category":   completeStatic(categoryValues()),
		"type":       completeStatic([]string{datamodel.TypeTicket, datamodel.TypeEpic}),
		"epic":       completeItems(g, datamodel.TypeEpic),
		"parent":     completeItems(g, datamodel.TypeEpic),
		"blocked-by": items,
		"board":      completeVocab(g, activeBoardKeys),
	}
	walk(root, func(c *cobra.Command) {
		for name, fn := range flags {
			if c.Flags().Lookup(name) != nil {
				_ = c.RegisterFlagCompletionFunc(name, fn)
			}
		}
		for _, lt := range datamodel.LinkTypes {
			if fl := core.FlagForLinkType(string(lt)); c.Flags().Lookup(fl) != nil {
				_ = c.RegisterFlagCompletionFunc(fl, items)
			}
		}
	})
}

func completeItems(g *globalFlags, typ string) completer {
	return func(_ *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		items, _, ok := cachedItems(g)
		if !ok {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		pref := strings.ToLower(toComplete)
		out := make([]string, 0, completionCap)
		for _, it := range items {
			if typ != "" && it.Type != typ {
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
		items, cfg, ok := cachedItems(g)
		if !ok {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		it := findCached(items, args[0])
		if it == nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return prefixed(core.MoveTargets(cfg, it.Type, it.State), toComplete), cobra.ShellCompDirectiveNoFileComp
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

func cachedItems(g *globalFlags) ([]*datamodel.Item, *datamodel.Config, bool) {
	s, cfg, err := openStore(g)
	if err != nil {
		return nil, nil, false
	}
	items, err := s.CachedItems()
	if err != nil {
		return nil, nil, false
	}
	return items, cfg, true
}

func findCached(items []*datamodel.Item, ref string) *datamodel.Item {
	for _, it := range items {
		if it.Number == ref || it.ID == ref || slices.Contains(it.Aliases, ref) {
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

func workflowTypes(cfg *datamodel.Config) []string {
	types := make([]string, 0, len(cfg.Workflows))
	for typ := range cfg.Workflows {
		types = append(types, typ)
	}
	slices.Sort(types)
	return types
}

func workflowStates(cfg *datamodel.Config) []string {
	seen := map[string]bool{}
	var out []string
	for _, wf := range cfg.Workflows {
		for _, st := range wf.States {
			if !seen[st.Key] {
				seen[st.Key] = true
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
