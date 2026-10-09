package models

import (
	"reflect"
	"strings"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

// init registers the custom "safetext" and "notblank" validation tags with Gin's
// validator when the package loads, so request structs can use them in
// binding tags.
//
// Arguments: None
//
// Returns: None
func init() {
	v, ok := binding.Validator.Engine().(*validator.Validate)
	if !ok {
		return
	}
	for tag, fn := range map[string]validator.Func{"safetext": safeText, "notblank": notBlank} {
		if err := v.RegisterValidation(tag, fn); err != nil {
			panic(err)
		}
	}
}

// notBlank rejects names that look filled in but are blank
//
// Arguments: fl (validator.FieldLevel) - the string field being validated
//
// Returns: bool - false if the string is empty or only whitespace
func notBlank(fl validator.FieldLevel) bool {
	f := fl.Field()
	if f.Kind() == reflect.Ptr {
		if f.IsNil() {
			return true
		}
		f = f.Elem()
	}
	if f.Kind() != reflect.String {
		return true
	}
	return strings.TrimSpace(f.String()) != ""
}

// safeText rejects text Postgres cannot store (NUL) and control characters that have
// no place in names or descriptions
//
// Arguments: fl (validator.FieldLevel) - the string field being validated
//
// Returns: bool - true unless the string contains a NUL byte or a control character other than tab, newline, or carriage return
func safeText(fl validator.FieldLevel) bool {
	f := fl.Field()
	if f.Kind() == reflect.Ptr {
		if f.IsNil() {
			return true
		}
		f = f.Elem()
	}
	if f.Kind() != reflect.String {
		return true
	}
	for _, r := range f.String() {
		if r < 0x20 && r != '\t' && r != '\n' && r != '\r' {
			return false
		}
		if r == 0x7f {
			return false
		}
	}
	return true
}
