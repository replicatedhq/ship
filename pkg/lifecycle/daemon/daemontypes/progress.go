package daemontypes

import (
	"encoding/json"
	"fmt"
	"sync"
)

type Progress struct {
	Source string `json:"source"`
	Type   string `json:"type"`  // string, json, etc
	Level  string `json:"level"` // string, json, etc
	Detail string `json:"detail,omitempty"`
}

func (p Progress) String() string {
	asJSON, err := json.Marshal(p)
	if err == nil {
		return string(asJSON)
	}
	return fmt.Sprintf("Progress{%s %s %s %s}", p.Source, p.Type, p.Level, p.Detail)
}

// Status parses the status code within progress detail that is used by the FE to determine
// progress phases within a step
func (p Progress) Status() string {
	if p.Detail != "" {
		var statusDetail map[string]string
		_ = json.Unmarshal([]byte(p.Detail), &statusDetail)
		return statusDetail["status"]
	}
	return ""
}

func StringProgress(source, detail string) Progress {
	return JSONProgress(source, map[string]interface{}{
		"status": detail,
	})
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

func MessageProgress(source string, msg Message) Progress {
	d, _ := json.Marshal(msg)
	return Progress{
		Source: source,
		Type:   "string",
		Level:  "info",
		Detail: string(d),
	}
}

// the empty value is initialized and ready to use
type ProgressMap struct {
	Map sync.Map
}

func (p *ProgressMap) Load(stepID string) (Progress, bool) {
	empty := Progress{}
	value, ok := p.Map.Load(stepID)
	if !ok {
		return empty, false
	}

	progress, ok := value.(Progress)
	if !ok {
		return empty, false
	}

	return progress, true
}

func (p *ProgressMap) Store(stepID string, progress Progress) {
	p.Map.Store(stepID, progress)
}

func (p *ProgressMap) Delete(stepID string) {
	p.Map.Delete(stepID)
}
