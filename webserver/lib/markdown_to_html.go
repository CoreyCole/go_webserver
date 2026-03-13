package lib

import (
	"errors"
	"fmt"
	"io"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/ast"
	mdhtml "github.com/gomarkdown/markdown/html"
	"github.com/rs/zerolog/log"
)

type MarkdownToHTMLRenderer struct {
	lightStyle     *chroma.Style
	darkStyle      *chroma.Style
	htmlFormatter  *html.Formatter
	mdhtmlRenderer *mdhtml.Renderer
}

func NewMarkdownToHtmlRenderer() (*MarkdownToHTMLRenderer, error) {
	lightStyle := styles.Get("github")
	darkStyle := styles.Get("github-dark")
	htmlFormatter := html.New(html.TabWidth(2))
	if htmlFormatter == nil {
		return nil, errors.New("couldn't create html formatter")
	}
	mdhtmlRenderer := mdhtmlRenderer(lightStyle, darkStyle, htmlFormatter)
	return &MarkdownToHTMLRenderer{
		lightStyle,
		darkStyle,
		htmlFormatter,
		mdhtmlRenderer,
	}, nil
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
) error {
	defaultLang := ""
	lang := string(codeBlock.Info)
	return htmlHighlight(
		w,
		string(codeBlock.Literal),
		lang,
		defaultLang,
		highlightStyle,
		htmlFormatter,
	)
}

func mdhtmlRenderer(
	lightStyle *chroma.Style,
	darkStyle *chroma.Style,
	htmlFormatter *html.Formatter,
) *mdhtml.Renderer {
	opts := mdhtml.RendererOptions{
		Flags: mdhtml.CommonFlags | mdhtml.HrefTargetBlank,
		RenderNodeHook: func(w io.Writer, node ast.Node, entering bool) (ast.WalkStatus, bool) {
			if code, ok := node.(*ast.CodeBlock); ok {
				// Light mode code block (hidden until Datastar shows it)
				w.Write(
					[]byte(
						`<div style="display:none" data-show="$theme !== 'dark'" class="my-4 rounded-xl shadow-lg [&>pre]:p-4">`,
					),
				)
				if err := renderCode(w, code, lightStyle, htmlFormatter); err != nil {
					log.Error().Msg("error rendering code")
					return ast.Terminate, false
				}
				w.Write([]byte("</div>"))
				// Dark mode code block (hidden until Datastar shows it)
				w.Write(
					[]byte(
						`<div style="display:none" data-show="$theme === 'dark'" class="my-4 rounded-xl shadow-lg [&>pre]:p-4">`,
					),
				)
				if err := renderCode(w, code, darkStyle, htmlFormatter); err != nil {
					log.Error().Msg("error rendering code")
					return ast.Terminate, false
				}
				w.Write([]byte("</div>"))
				return ast.GoToNext, true
			}
			if link, ok := node.(*ast.Link); ok {
				if entering {
					w.Write(
						[]byte(
							`<a class="font-medium text-primary hover:underline"`,
						),
					)
					if len(link.Title) > 0 {
						w.Write(
							[]byte(fmt.Sprintf(` title="link" href="%s"`, link.Title)),
						)
					} else if len(link.Destination) > 0 {
						w.Write([]byte(fmt.Sprintf(` href="%s"`, link.Destination)))
					}
					w.Write([]byte(">"))
				} else {
					w.Write([]byte("</a>"))
				}
				return ast.GoToNext, true
			}
			if heading, ok := node.(*ast.Heading); ok {
				headingClass := fmt.Sprintf(
					"myh myh-%d",
					min(heading.Level, 6), // max supported level is 6
				)

				if headingClass != "" {
					attr := heading.Attribute
					if attr == nil {
						attr = &ast.Attribute{}
					}
					attr.Classes = append(
						attr.Classes,
						[]byte(headingClass),
					)
					heading.Attribute = attr
				}
			}
			if list, ok := node.(*ast.List); ok {
				listClass := "list-disc"
				if list.ListFlags&ast.ListTypeOrdered != 0 {
					listClass = "list-decimal"
				}
				if entering {
					// Start of the list
					fmt.Fprintf(w, `<ul class="%s ml-6">`, listClass)
				} else {
					// End of the list
					fmt.Fprintf(w, "</ul>")
				}
				return ast.GoToNext, true
			}
			if _, ok := node.(*ast.ListItem); ok {
				if entering {
					w.Write([]byte(`<li class="text-lg">`))
				} else {
					w.Write([]byte("</li>"))
				}
				return ast.GoToNext, true
			}
			if p, ok := node.(*ast.Paragraph); ok {
				attr := p.Attribute
				if attr == nil {
					attr = &ast.Attribute{}
				}
				attr.Classes = append(attr.Classes, []byte("text-lg"))
				p.Attribute = attr
			}

			// return false to tell html.Renderer to use default render
			return ast.GoToNext, false
		},
	}
	return mdhtml.NewRenderer(opts)
}
