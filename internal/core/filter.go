package core

import (
	"errors"
	"maps"
	"slices"
	"strings"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/query"
)

func filterNames(cfg *datamodel.Config) []string {
	return slices.Sorted(maps.Keys(cfg.Filters))
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

func queryError(err error) error {
	var qerr *query.Error
	if !errors.As(err, &qerr) {
		return err
	}
	return errx.User("%w", err).WithHint("check the query expression syntax")
}
