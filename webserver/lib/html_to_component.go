package lib

import (
	"context"
	"html/template"
	"io"

	"github.com/a-h/templ"
)

func HTMLToComponent(htmlString string) templ.Component {
	return templ.ComponentFunc(func(_ context.Context, w io.Writer) error {
		// Convert the raw HTML string to template.HTML to prevent it from being escaped.
		safeHTML := template.HTML(htmlString)
		_, err := io.WriteString(w, string(safeHTML))
		return err
	})
}
