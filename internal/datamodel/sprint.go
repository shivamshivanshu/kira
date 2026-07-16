package datamodel

type Sprint struct {
	Key   string `yaml:"key"`
	Name  string `yaml:"name"`
	Start string `yaml:"start"`
	End   string `yaml:"end"`
}

func (c *Config) HasSprint(key string) bool {
	for _, s := range c.Sprints {
		if s.Key == key {
			return true
		}
	}
	return false
}
