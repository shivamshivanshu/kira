package core

import (
	"github.com/shivamshivanshu/kira/internal/config"
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

func (s *Store) ConfigSet(cfg *datamodel.Config, key, value string) (*datamodel.ConfigSetResult, error) {
	err := s.mutateConfig(func(data []byte, _ *datamodel.Config) (configEdit, error) {
		out, err := config.SetScalar(data, key, value)
		if err != nil {
			return configEdit{}, errx.User("%v", err)
		}
		return configEdit{data: out, commit: cfg.Commit, subject: "config set " + key}, nil
	})
	if err != nil {
		return nil, err
	}
	return &datamodel.ConfigSetResult{Key: key, Value: value}, nil
}
