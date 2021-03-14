package component

type TCommand struct {
	Service    string `json:"service"`
	SQL        string `json:"sql"`
	Collection string `json:"collection"`
	Filter     *struct {
		Field string `json:"field"`
		Query string `json:"query"`
	} `json:"filter"`
}
