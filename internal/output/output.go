package output

import (
	"io"
	"os"
)

// JSON marshals v as compact JSON and writes it to stdout with a trailing newline.
func JSON(v any) error {
	return JSONTo(os.Stdout, v)
}

// JSONTo marshals v as compact JSON and writes it to w with a trailing newline.
func JSONTo(w io.Writer, v any) error {
	return nil // stub - tests should fail
}
