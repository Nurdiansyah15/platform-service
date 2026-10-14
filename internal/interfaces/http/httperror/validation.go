package httperror

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

// RegisterValidatorTagNames membuat validator memakai nama dari tag json/form/uri,
// sehingga key error validasi sama dengan nama field di request.
func RegisterValidatorTagNames() {
	v, ok := binding.Validator.Engine().(*validator.Validate)
	if !ok {
		return
	}
	v.RegisterTagNameFunc(func(f reflect.StructField) string {
		for _, tag := range []string{"json", "form", "uri"} {
			name := strings.SplitN(f.Tag.Get(tag), ",", 2)[0]
			if name == "-" {
				return ""
			}
			if name != "" {
				return name
			}
		}
		return f.Name
	})
}

// ParseValidationErrors memetakan validator.ValidationErrors ke map field → pesan.
func ParseValidationErrors(locale string, err validator.ValidationErrors) map[string]string {
	out := make(map[string]string, len(err))
	for _, e := range err {
		out[e.Field()] = translateValidationTag(locale, e.Tag(), e.Param())
	}
	return out
}

func translateValidationTag(locale, tag, param string) string {
	key := "validation." + tag
	msg := translate(locale, key)
	if msg == key {
		msg = translate(locale, "validation.default")
		if msg == "validation.default" {
			return "Invalid value"
		}
	}
	if strings.Contains(msg, "%s") && param != "" {
		return fmt.Sprintf(msg, param)
	}
	return msg
}
