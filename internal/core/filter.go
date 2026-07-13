package core

import (
	"sort"
	"strings"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
)

func Filters(cfg *datamodel.Config) *datamodel.FilterListResult {
	views := make([]datamodel.FilterView, 0, len(cfg.Filters))
	for _, name := range filterNames(cfg) {
		views = append(views, datamodel.FilterView{Name: name, Query: cfg.Filters[name]})
	}
	return &datamodel.FilterListResult{Filters: views}
}

func filterNames(cfg *datamodel.Config) []string {
	names := make([]string, 0, len(cfg.Filters))
	for name := range cfg.Filters {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func unknownFilterErr(cfg *datamodel.Config, name string) error {
	if len(cfg.Filters) == 0 {
		return errx.User("unknown filter %q (no filters configured)", name).WithHint("define filters under `filters:` in .kira/config.yaml")
	}
	names := filterNames(cfg)
	base := errx.User("unknown filter %q (available: %s)", name, strings.Join(names, ", "))
	if n := errx.Nearest(name, names); n != "" {
		return base.WithHint("did you mean `%s`?", n)
	}
	return base
}
