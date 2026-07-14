package core

import (
	"strings"

	"github.com/shivamshivanshu/kira/internal/datamodel"
	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/id"
)

func boardKeyOf(number string) string {
	return id.KeyOf(number)
}

func resolveBoardKey(cfg *datamodel.Config, explicit string) (string, error) {
	boards := cfg.ActiveBoards()
	if len(boards) == 0 {
		if len(cfg.EffectiveBoards()) > 0 {
			return "", errx.User("all boards are archived").
				WithHint("unarchive one: kira board unarchive <KEY>")
		}
		return "", errx.User("no boards configured").
			WithHint(`create one first: kira board create ABC --name "Alpha"`)
	}
	if explicit != "" {
		b, ok := cfg.BoardByKey(explicit)
		if !ok || b.Archived {
			return "", errx.User("no such board %q", explicit).
				WithHint("boards: %s", strings.Join(activeBoardKeys(boards), ", "))
		}
		return b.Key, nil
	}
	if len(boards) == 1 {
		return boards[0].Key, nil
	}
	if b, ok := cfg.DefaultBoard(); ok {
		return b.Key, nil
	}
	return "", errMultipleBoards(boards)
}

func errMultipleBoards(boards []datamodel.Board) error {
	return errx.User("multiple boards configured; pass --board").
		WithHint("boards: %s — or mark one default in .kira/config.yaml", strings.Join(activeBoardKeys(boards), ", "))
}

func activeBoardKeys(boards []datamodel.Board) []string {
	keys := make([]string, len(boards))
	for i, b := range boards {
		keys[i] = b.Key
	}
	return keys
}
