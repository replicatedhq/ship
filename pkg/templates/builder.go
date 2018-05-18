package templates

/*
  This was taken from https://github.com/replicatedcom/replicated/blob/master/templates/builder.go
*/

import (
	"bytes"
	"os"
	"regexp"
	"strconv"
	"text/template"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

var (
	templateNotDefinedRegexp = regexp.MustCompile(`template.*not defined$`)
)

type Builder struct {
	Ctx    []Ctx
	Functs template.FuncMap
	Logger log.Logger
}

func NewBuilder(ctxx ...Ctx) Builder {
	var builder Builder

	builder.Logger = log.NewLogfmtLogger(os.Stderr)
	for _, ctx := range ctxx {
		builder.AddCtx(ctx)
	}

	return builder
}

func (b *Builder) AddCtx(ctx Ctx) {
	b.Ctx = append(b.Ctx, ctx)
}

func (b *Builder) String(text string) (string, error) {
	if text == "" {
		return "", nil
	}
	return b.RenderTemplate(text, text)
}

func (b *Builder) Bool(text string, defaultVal bool) (bool, error) {
	if text == "" {
		return defaultVal, nil
	}

	value, err := b.RenderTemplate(text, text)
	if err != nil {
		return defaultVal, err
	}

	// If the template didn't parse (turns into an empty string), then we should
	// return the default
	if value == "" {
		return defaultVal, nil
	}

	result, err := strconv.ParseBool(value)
	if err != nil {
		level.Error(b.Logger).Log("msg", "Template builder failed to parse bool", "value", value, "err", err)
		// for now we are assuming default value if we fail to parse
		return defaultVal, nil
	}

	return result, nil
}

func (b *Builder) Int(text string, defaultVal int64) (int64, error) {
	if text == "" {
		return defaultVal, nil
	}

	value, err := b.RenderTemplate(text, text)
	if err != nil {
		return defaultVal, err
	}

	// If the template didn't parse (turns into an empty string), then we should
	// return the default
	if value == "" {
		return defaultVal, nil
	}

	result, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		level.Error(b.Logger).Log("msg", "Template builder failed to parse int: %v", err)
		// for now we are assuming default value if we fail to parse
		return defaultVal, nil
	}

	return result, nil
}

func (b *Builder) Uint(text string, defaultVal uint64) (uint64, error) {
	if text == "" {
		return defaultVal, nil
	}

	value, err := b.RenderTemplate(text, text)
	if err != nil {
		return defaultVal, err
	}

	// If the template didn't parse (turns into an empty string), then we should
	// return the default
	if value == "" {
		return defaultVal, nil
	}

	result, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		level.Error(b.Logger).Log("msg", "Template builder failed to parse int: %v", err)
		// for now we are assuming default value if we fail to parse
		return defaultVal, nil
	}

	return result, nil
}

func (b *Builder) Float64(text string, defaultVal float64) (float64, error) {
	if text == "" {
		return defaultVal, nil
	}

	value, err := b.RenderTemplate(text, text)
	if err != nil {
		return defaultVal, err
	}

	// If the template didn't parse (turns into an empty string), then we should
	// return the default
	if value == "" {
		return defaultVal, nil
	}

	result, err := strconv.ParseFloat(value, 64)
	if err != nil {
		level.Error(b.Logger).Log("msg", "Template builder failed to parse float64: %v", err)
		// for now we are assuming default value if we fail to parse
		return defaultVal, nil
	}

	return result, nil
}

func (b *Builder) BuildFuncMap() template.FuncMap {
	if b.Functs == nil {
		b.Functs = template.FuncMap{}
	}

	funcMap := b.Functs
	for _, ctx := range b.Ctx {
		for name, fn := range ctx.FuncMap() {
			funcMap[name] = fn
		}
	}
	return funcMap
}

func (b *Builder) GetTemplate(name, text string) (*template.Template, error) {
	tmpl, err := template.New(name).Delims("{{repl ", "}}").Funcs(b.BuildFuncMap()).Parse(text)
	if err != nil {
		if !templateNotDefinedRegexp.MatchString(err.Error()) {
			level.Error(b.Logger).Log("msg", err)
		}
		return nil, err
	}
	return tmpl, nil
}

func (b *Builder) RenderTemplate(name string, text string) (string, error) {
	tmpl, err := b.GetTemplate(name, text)
	if err != nil {
		return "", err
	}
	var contents bytes.Buffer
	if err := tmpl.Execute(&contents, nil); err != nil {
		level.Error(b.Logger).Log("msg", err)
		return "", err
	}
	return contents.String(), nil
}
