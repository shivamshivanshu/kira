package tui

type chord struct {
	pending string
}

func (c *chord) arm(key string) { c.pending = key }

func (c *chord) take() (string, bool) {
	p := c.pending
	c.pending = ""
	return p, p != ""
}
