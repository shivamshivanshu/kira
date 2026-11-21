package core

import (
	_ "embed"
	"fmt"

	"github.com/shivamshivanshu/kira/internal/codec"
	"github.com/shivamshivanshu/kira/internal/config"
	"github.com/shivamshivanshu/kira/internal/datamodel"
)

//go:embed templates/repo-config.yaml
var configTemplate string

func initConfigYAML(key, name string) (string, error) {
	defaultBoard, err := config.InlineBoardEntry(datamodel.Board{Key: key, Name: name, Default: true})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(configTemplate, codec.EmitScalar(key), codec.EmitScalar(name), defaultBoard), nil
}
