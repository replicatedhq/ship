package daemon

import "encoding/json"

type Progress struct {
	Source string `json:"source"`
	Type   string `json:"type"`  // string, json, etc
	Level  string `json:"level"` // string, json, etc
	Detail string `json:"detail,omitempty"`
}

func StringProgress(source, detail string) Progress {
	return Progress{
		Source: source,
		Type:   "string",
		Level:  "info",
		Detail: detail,
	}
}

func JSONProgress(source string, detail interface{}) Progress {
	d, _ := json.Marshal(detail)
	return Progress{
		Source: source,
		Type:   "json",
		Level:  "info",
		Detail: string(d),
	}
}
