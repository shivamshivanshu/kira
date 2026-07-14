package config

import (
	"reflect"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/shivamshivanshu/kira/internal/datamodel"
)

func uncomment(template string) string {
	lines := strings.Split(template, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "# ") {
			lines[i] = line[2:]
		}
	}
	return strings.Join(lines, "\n")
}

func TestUserConfigTemplateMatchesDefaults(t *testing.T) {
	t.Parallel()
	var got struct {
		UI     datamodel.UI     `yaml:"ui"`
		Workon datamodel.Workon `yaml:"workon"`
	}
	if err := yaml.Unmarshal([]byte(uncomment(mustTemplate(userConfigFileName))), &got); err != nil {
		t.Fatalf("uncommented template is not valid YAML: %v", err)
	}

	if err := validateUISection(got.UI); err != nil {
		t.Fatalf("template ui section is invalid: %v", err)
	}
	if w := UIWarnings(got.UI); len(w) != 0 {
		t.Fatalf("template ui section raised warnings: %v", w)
	}

	def := Default()
	wantUI := def.UI
	wantUI.Theme, got.UI.Theme = nil, nil
	if !reflect.DeepEqual(got.UI, wantUI) {
		t.Errorf("template ui defaults drifted:\n got  %+v\n want %+v", got.UI, wantUI)
	}

	wantWorkon := def.Workon
	wantWorkon.BranchPattern, wantWorkon.Casing = "", ""
	if !reflect.DeepEqual(got.Workon, wantWorkon) {
		t.Errorf("template workon defaults drifted:\n got  %+v\n want %+v", got.Workon, wantWorkon)
	}
}

func TestUserHooksTemplateIsValid(t *testing.T) {
	t.Parallel()
	var hooks []datamodel.AutomationHook
	if err := yaml.Unmarshal([]byte(uncomment(mustTemplate(userHooksYAMLName))), &hooks); err != nil {
		t.Fatalf("uncommented hooks template is not valid YAML: %v", err)
	}
	if len(hooks) == 0 {
		t.Fatal("hooks template must carry an example hook")
	}
	if err := validateAutomationHooks("hooks", hooks); err != nil {
		t.Fatalf("hooks template example is invalid: %v", err)
	}
}
