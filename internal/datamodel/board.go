package datamodel

type BoardColumn struct {
	State    string     `json:"state"`
	Category string     `json:"category"`
	Wip      int        `json:"wip"`
	Count    int        `json:"count"`
	Items    []ListItem `json:"items"`
}

type BoardResult struct {
	Type    string        `json:"type"`
	Columns []BoardColumn `json:"columns"`
}

func (r *BoardResult) Empty() bool {
	for _, c := range r.Columns {
		if len(c.Items) > 0 {
			return false
		}
	}
	return true
}
