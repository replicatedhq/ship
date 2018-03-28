package lifecycle

import (
	"context"

	"bytes"
	"text/template"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/mitchellh/cli"
	"github.com/pkg/errors"
	"github.com/replicatedcom/ship/pkg/api"
	"github.com/replicatedcom/ship/pkg/lifecycle/render"
	"github.com/spf13/viper"
)

type messenger struct {
	Logger log.Logger
	UI     cli.Ui
	Viper  *viper.Viper
}

func (e *messenger) Execute(ctx context.Context, step *api.Message) error {
	debug := level.Debug(log.With(e.Logger, "step.type", "message"))

	debug.Log("event", "step.execute", "step.level", step.Level)

	tpl, err := template.New("message step").
		Delims("{{ship ", "}}").
		Funcs(e.funcMap()).
		Parse(step.Contents)
	if err != nil {
		return errors.Wrapf(err, "Parse template for message at %s", step.Contents)
	}

	var rendered bytes.Buffer
	err = tpl.Execute(&rendered, nil)
	if err != nil {
		return errors.Wrapf(err, "Execute template for message at %s", step.Contents)
	}

	switch step.Level {
	case "error":
		e.UI.Error(rendered.String())
	case "warn":
		e.UI.Warn(rendered.String())
	case "debug":
		e.UI.Output(rendered.String())
	default:
		e.UI.Info(rendered.String())
	}
	return nil
}

func (e *messenger) funcMap() template.FuncMap {
	debug := level.Debug(log.With(e.Logger, "step.type", "render", "render.phase", "template"))

	return map[string]interface{}{
		"config": func(name string) interface{} {
			configItemValue := e.Viper.Get(name)
			if configItemValue == "" {
				debug.Log("event", "template.missing", "func", "config", "requested", name)
				return ""
			}
			return configItemValue
		},
		"context": func(name string) interface{} {
			switch name {
			case "state_file_path":
				return render.StateFilePath
			}
			debug.Log("event", "template.missing", "func", "context", "requested", name)
			return ""
		},
	}
}
