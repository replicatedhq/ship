package config

import "encoding/json"

type Progress struct {
	Source string `json:"source"`
	Type   string `json:"type"` // string, json, etc
	Detail string `json:"detail,omitempty"`
}

func StringProgress(source, detail string) Progress {
	return Progress{
		Source: source,
		Type:   "string",
		Detail: detail,
	}
}

func JSONProgress(source string, detail interface{}) Progress {
	d, _ := json.Marshal(detail)
	return Progress{
		Source: source,
		Type:   "json",
		Detail: string(d),
	}
}
