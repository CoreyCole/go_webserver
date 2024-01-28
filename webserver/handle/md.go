package handle

import (
	"context"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"

	"github.com/a-h/templ"
	"github.com/alecthomas/chroma"
	"github.com/alecthomas/chroma/formatters/html"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/styles"
	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/ast"
	mdhtml "github.com/gomarkdown/markdown/html"
	"github.com/labstack/echo"

	"github.com/coreycole/go_md/webserver/views"
)

var (
	htmlFormatter  *html.Formatter
	highlightStyle *chroma.Style
)

func init() {
	htmlFormatter = html.New(html.Standalone(true), html.TabWidth(2))
	if htmlFormatter == nil {
		panic("couldn't create html formatter")
	}
	styleName := "monokai"
	highlightStyle = styles.Get(styleName)
	if highlightStyle == nil {
		panic(fmt.Sprintf("didn't find style '%s'", styleName))
	}
}

// based on https://github.com/alecthomas/chroma/blob/master/quick/quick.go
func htmlHighlight(w io.Writer, source, lang, defaultLang string) error {
	if lang == "" {
		lang = defaultLang
	}
	l := lexers.Get(lang)
	if l == nil {
		l = lexers.Analyse(source)
	}
	if l == nil {
		l = lexers.Fallback
	}
	l = chroma.Coalesce(l)

	it, err := l.Tokenise(nil, source)
	if err != nil {
		return err
	}
	return htmlFormatter.Format(w, highlightStyle, it)
}

// an actual rendering of Paragraph is more complicated
func renderCode(w io.Writer, codeBlock *ast.CodeBlock, _ bool) {
	defaultLang := ""
	lang := string(codeBlock.Info)
	htmlHighlight(w, string(codeBlock.Literal), lang, defaultLang)
}

func myRenderHook(w io.Writer, node ast.Node, entering bool) (ast.WalkStatus, bool) {
	if code, ok := node.(*ast.CodeBlock); ok {
		renderCode(w, code, entering)
		return ast.GoToNext, true
	}
	return ast.GoToNext, false
}

func newCustomizedRender() *mdhtml.Renderer {
	opts := mdhtml.RendererOptions{
		Flags:          mdhtml.CommonFlags,
		RenderNodeHook: myRenderHook,
	}
	return mdhtml.NewRenderer(opts)
}

func HTMLContent(rawHTML string) templ.Component {
	return templ.ComponentFunc(func(_ context.Context, w io.Writer) error {
		// Convert the raw HTML string to template.HTML to prevent it from being escaped.
		safeHTML := template.HTML(rawHTML)
		_, err := io.WriteString(w, string(safeHTML))
		return err
	})
}

func ServeMarkdown(c echo.Context) error {
	filename := c.Param("filename")

	// Read markdown file
	md, err := os.ReadFile("md/" + filename)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "File not found")
	}

	// Convert markdown to HTML
	renderer := newCustomizedRender()
	htmlBytes := markdown.ToHTML(md, nil, renderer)

	// Convert HTML bytes to a string
	htmlString := string(htmlBytes)
	htmlComponent := HTMLContent(htmlString)

	// Use the Page templ component to construct the full page HTML
	view := views.MarkdownPage(htmlComponent)

	if err := view.Render(c.Request().Context(), c.Response().Writer); err != nil {
		return echo.NewHTTPError(
			http.StatusInternalServerError,
			"Error rendering page: "+err.Error(),
		)
	}

	return nil
}
