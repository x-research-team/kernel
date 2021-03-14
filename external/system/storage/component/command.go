package component

type TCommand struct {
	Service    string                 `json:"service"`
	SQL        string                 `json:"sql"`
	Collection string                 `json:"collection"`
	Filter     map[string]interface{} `json:"filter"`
}
