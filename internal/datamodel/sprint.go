package datamodel

type Sprint struct {
	Key   string `yaml:"key"`
	Name  string `yaml:"name"`
	Start string `yaml:"start"`
	End   string `yaml:"end"`
}

func (c *Config) Sprint(key string) (Sprint, bool) {
	for _, s := range c.Sprints {
		if s.Key == key {
			return s, true
		}
	}
	return Sprint{}, false
}

func (c *Config) HasSprint(key string) bool {
	_, ok := c.Sprint(key)
	return ok
}
