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

// fieldFilterErr reframes a grammar error as a `--field key=value` error:
// unlike a native flag, both halves of that pair are typed freehand, so the
// key gets a did-you-mean and nothing should mention query syntax positions.
func fieldFilterErr(key string, err error, extras []query.ExtraField) error {
	if query.IsUnknownField(err) {
		names := query.FieldNames(extras)
		base := errx.User("--field %q: unknown field (available: %s)", key, strings.Join(names, ", "))
		if n := errx.Nearest(key, names); n != "" {
			return base.WithHint("did you mean `%s`?", n)
		}
		return base
	}
	if qe, ok := errors.AsType[*query.Error](err); ok {
		return errx.User("--field %q: %s", key, qe.Msg)
	}
	return err
}

func queryError(err error) error {
	if _, ok := errors.AsType[*query.Error](err); !ok {
		return err
	}
	return errx.User("%w", err).WithHint("check the query expression syntax")
}
