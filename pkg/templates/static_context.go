package templates

/*
  This was taken from https://github.com/replicatedcom/replicated/blob/master/templates/context.go
*/

import (
	"encoding/base64"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"strings"
	"text/template"
	"time"

	units "github.com/docker/go-units"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

func NewStaticContext() *StaticCtx {
	staticCtx := &StaticCtx{
		Logger: log.NewLogfmtLogger(os.Stderr),
	}
	return staticCtx
}

type Ctx interface {
	FuncMap() template.FuncMap
}

type StaticCtx struct {
	Logger log.Logger
}

func (ctx StaticCtx) FuncMap() template.FuncMap {
	return template.FuncMap{
		"Now":          ctx.now,
		"NowFmt":       ctx.nowFormat,
		"ToLower":      strings.ToLower,
		"ToUpper":      strings.ToUpper,
		"TrimSpace":    strings.TrimSpace,
		"Trim":         ctx.trim,
		"UrlEncode":    url.QueryEscape,
		"Base64Encode": ctx.base64Encode,
		"Base64Decode": ctx.base64Decode,
		"Split":        strings.Split,
		"RandomString": ctx.RandomString,
		"Add":          ctx.add,
		"Sub":          ctx.sub,
		"Mult":         ctx.mult,
		"Div":          ctx.div,
		"ParseBool":    ctx.parseBool,
		"ParseFloat":   ctx.parseFloat,
		"ParseInt":     ctx.parseInt,
		"ParseUint":    ctx.parseUint,
		"HumanSize":    ctx.humanSize,
	}
}

func (ctx StaticCtx) now() string {
	return ctx.nowFormat("")
}

func (ctx StaticCtx) nowFormat(format string) string {
	if format == "" {
		format = time.RFC3339
	}
	return time.Now().UTC().Format(format)
}

func (ctx StaticCtx) trim(s string, args ...string) string {
	if len(args) == 0 {
		return strings.TrimSpace(s)
	}
	return strings.Trim(s, args[0])
}

func (ctx StaticCtx) base64Encode(plain string) string {
	return base64.StdEncoding.EncodeToString([]byte(plain))
}

func (ctx StaticCtx) base64Decode(encoded string) string {
	plain, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		level.Error(ctx.Logger).Log("msg", "unable to base64 decode", "err", err)
		return ""
	}
	return string(plain)
}

func (ctx StaticCtx) add(a, b interface{}) interface{} {
	av := reflect.ValueOf(a)
	bv := reflect.ValueOf(b)

	if ctx.isFloat(av) || ctx.isFloat(bv) {
		return ctx.reflectToFloat(av) + ctx.reflectToFloat(bv)
	}
	if ctx.isInt(av) {
		return av.Int() + ctx.reflectToInt(bv)
	}
	if ctx.isUint(av) {
		return av.Uint() + ctx.reflectToUint(bv)
	}
	level.Error(ctx.Logger).Log("msg", "unable to add")
	return 0
}

func (ctx StaticCtx) sub(a, b interface{}) interface{} {
	av := reflect.ValueOf(a)
	bv := reflect.ValueOf(b)

	if ctx.isFloat(av) || ctx.isFloat(bv) {
		return ctx.reflectToFloat(av) - ctx.reflectToFloat(bv)
	}
	if ctx.isInt(av) {
		return av.Int() - ctx.reflectToInt(bv)
	}
	if ctx.isUint(av) {
		return av.Uint() - ctx.reflectToUint(bv)
	}
	level.Error(ctx.Logger).Log("msg", "unable to sub")
	return 0
}

func (ctx StaticCtx) mult(a, b interface{}) interface{} {
	av := reflect.ValueOf(a)
	bv := reflect.ValueOf(b)

	if ctx.isFloat(av) || ctx.isFloat(bv) {
		return ctx.reflectToFloat(av) * ctx.reflectToFloat(bv)
	}
	if ctx.isInt(av) {
		return av.Int() * ctx.reflectToInt(bv)
	}
	if ctx.isUint(av) {
		return av.Uint() * ctx.reflectToUint(bv)
	}
	level.Error(ctx.Logger).Log("msg", "unable to mult")
	return 0
}

func (ctx StaticCtx) div(a, b interface{}) interface{} {
	av := reflect.ValueOf(a)
	bv := reflect.ValueOf(b)

	if ctx.isFloat(av) || ctx.isFloat(bv) {
		return ctx.reflectToFloat(av) / ctx.reflectToFloat(bv)
	}
	if ctx.isInt(av) {
		return av.Int() / ctx.reflectToInt(bv)
	}
	if ctx.isUint(av) {
		return av.Uint() / ctx.reflectToUint(bv)
	}
	level.Error(ctx.Logger).Log("msg", "unable to div")
	return 0
}

func (ctx StaticCtx) parseBool(str string) bool {
	val, err := strconv.ParseBool(str)
	if err != nil {
		level.Error(ctx.Logger).Log("msg", "unable to parseBool", "err", err)
	}
	return val
}

func (ctx StaticCtx) parseFloat(str string) float64 {
	val, err := strconv.ParseFloat(str, 64)
	if err != nil {
		level.Error(ctx.Logger).Log("msg", "unable to parseFloat", "err", err)
	}
	return val
}

func (ctx StaticCtx) parseInt(str string, args ...int) int64 {
	base := 10
	if len(args) > 0 {
		base = args[0]
	}
	val, err := strconv.ParseInt(str, base, 64)
	if err != nil {
		level.Error(ctx.Logger).Log("msg", "unable to parseInt", "err", err)
	}
	return val
}

func (ctx StaticCtx) parseUint(str string, args ...int) uint64 {
	base := 10
	if len(args) > 0 {
		base = args[0]
	}
	val, err := strconv.ParseUint(str, base, 64)
	if err != nil {
		level.Error(ctx.Logger).Log("msg", "unable to parseUint", "err", err)
	}
	return val
}

func (ctx StaticCtx) humanSize(size interface{}) string {
	v := reflect.ValueOf(size)
	return units.HumanSize(ctx.reflectToFloat(v))
}

func (ctx StaticCtx) reflectToFloat(val reflect.Value) float64 {
	if ctx.isFloat(val) {
		return val.Float()
	}
	if ctx.isInt(val) {
		return float64(val.Int())
	}
	if ctx.isUint(val) {
		return float64(val.Uint())
	}
	level.Error(ctx.Logger).Log("msg", "unable to convert to float")
	return 0
}

func (ctx StaticCtx) reflectToInt(val reflect.Value) int64 {
	if ctx.isFloat(val) {
		return int64(val.Float())
	}
	if ctx.isInt(val) || ctx.isUint(val) {
		return val.Int()
	}
	level.Error(ctx.Logger).Log("msg", "unable to convert to int")
	return 0
}

func (ctx StaticCtx) reflectToUint(val reflect.Value) uint64 {
	if ctx.isFloat(val) {
		return uint64(val.Float())
	}
	if ctx.isInt(val) || ctx.isUint(val) {
		return val.Uint()
	}
	level.Error(ctx.Logger).Log("msg", "unable to convert to uint")
	return 0
}

func (ctx StaticCtx) isFloat(val reflect.Value) bool {
	kind := val.Kind()
	return kind == reflect.Float32 || kind == reflect.Float64
}

func (ctx StaticCtx) isInt(val reflect.Value) bool {
	kind := val.Kind()
	return kind == reflect.Int || kind == reflect.Int8 || kind == reflect.Int16 || kind == reflect.Int32 || kind == reflect.Int64
}

func (ctx StaticCtx) isUint(val reflect.Value) bool {
	kind := val.Kind()
	return kind == reflect.Uint || kind == reflect.Uint8 || kind == reflect.Uint16 || kind == reflect.Uint32 || kind == reflect.Uint64
}
