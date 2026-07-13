package datamodel

type Renumbering struct {
	ID   string `json:"id"`
	From string `json:"from"`
	To   string `json:"to"`
}

type ReconcileResult struct {
	Renumbered []Renumbering `json:"renumbered"`
}
