package lib

import (
	"errors"
	"io"

	"github.com/alecthomas/chroma"
	"github.com/alecthomas/chroma/formatters/html"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/styles"
	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/ast"
	mdhtml "github.com/gomarkdown/markdown/html"
)

const (
	DEFAULT_CODE_STYLE = "monokai"
)

type MarkdownToHTMLRenderer struct {
	highlightStyle *chroma.Style
	htmlFormatter  *html.Formatter
	mdhtmlRenderer *mdhtml.Renderer
}

func NewMarkdownToHtmlRenderer(highlightStyleString string) (*MarkdownToHTMLRenderer, error) {
	var styleName string
	if highlightStyleString == "" {
		styleName = DEFAULT_CODE_STYLE
	} else {
		styleName = highlightStyleString
	}
	highlightStyle := styles.Get(DEFAULT_CODE_STYLE)
	if style, ok := styles.Registry[styleName]; ok {
		highlightStyle = style
	}
	htmlFormatter := html.New(html.Standalone(true), html.TabWidth(2))
	if htmlFormatter == nil {
		return nil, errors.New("couldn't create html formatter")
	}
	mdhtmlRenderer := mdhtmlRenderer(highlightStyle, htmlFormatter)
	return &MarkdownToHTMLRenderer{highlightStyle, htmlFormatter, mdhtmlRenderer}, nil
}

func (m MarkdownToHTMLRenderer) MarkdownBytesToHTML(md []byte) string {
	htmlBytes := markdown.ToHTML(md, nil, m.mdhtmlRenderer)
	return string(htmlBytes)
}

// based on https://github.com/alecthomas/chroma/blob/master/quick/quick.go
func htmlHighlight(
	w io.Writer,
	source, lang,
	defaultLang string,
	highlightStyle *chroma.Style,
	htmlFormatter *html.Formatter,
) error {
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

func renderCode(
	w io.Writer,
	codeBlock *ast.CodeBlock,
	highlightStyle *chroma.Style,
	htmlFormatter *html.Formatter,
) {
	defaultLang := ""
	lang := string(codeBlock.Info)
	htmlHighlight(w, string(codeBlock.Literal), lang, defaultLang, highlightStyle, htmlFormatter)
}

func mdhtmlRenderer(highlightStyle *chroma.Style, htmlFormatter *html.Formatter) *mdhtml.Renderer {
	opts := mdhtml.RendererOptions{
		Flags: mdhtml.CommonFlags,
		RenderNodeHook: func(w io.Writer, node ast.Node, _ bool) (ast.WalkStatus, bool) {
			if code, ok := node.(*ast.CodeBlock); ok {
				renderCode(w, code, highlightStyle, htmlFormatter)
				return ast.GoToNext, true
			}
			return ast.GoToNext, false
		},
	}
	return mdhtml.NewRenderer(opts)
}
