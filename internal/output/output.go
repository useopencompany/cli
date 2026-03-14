package output

import (
	"encoding/json"
	"io"
	"os"
	"reflect"
)

// JSON marshals v as compact JSON and writes it to stdout with a trailing newline.
func JSON(v any) error {
	return JSONTo(os.Stdout, v)
}

// JSONTo marshals v as compact JSON and writes it to w with a trailing newline.
func JSONTo(w io.Writer, v any) error {
	v = ensureNonNil(v)
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	_, err = w.Write(data)
	return err
}

// ensureNonNil checks if v is a nil slice and returns an empty slice instead,
// so that JSON serialization produces "[]" rather than "null".
func ensureNonNil(v any) any {
	if v == nil {
		return v
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Slice && rv.IsNil() {
		return []any{}
	}
	return v
}
