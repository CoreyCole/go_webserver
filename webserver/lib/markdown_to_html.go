package lib

import (
	"errors"
	"fmt"
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
	} else {
		fmt.Printf("invaid style: %s\n", styleName)
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

func mdhtmlRenderer(highlightStyle *chroma.Style, htmlFormatter *html.Formatter) *mdhtml.Renderer {
	opts := mdhtml.RendererOptions{
		Flags: mdhtml.CommonFlags | mdhtml.HrefTargetBlank,
		RenderNodeHook: func(w io.Writer, node ast.Node, entering bool) (ast.WalkStatus, bool) {
			if code, ok := node.(*ast.CodeBlock); ok {
				w.Write([]byte(`<div class="my-4 rounded-xl shadow-lg [&>pre]:p-4">`))
				err := renderCode(w, code, highlightStyle, htmlFormatter)
				if err != nil {
					fmt.Println("error rendering code")
					return ast.Terminate, false
				}
				w.Write([]byte("</div>"))
				return ast.GoToNext, true
			}
			if link, ok := node.(*ast.Link); ok {
				if entering {
					w.Write(
						[]byte(
							`<a class="font-medium text-green-600 dark:text-green-400 hover:underline"`,
						),
					)
					if len(link.Title) > 0 {
						w.Write([]byte(fmt.Sprintf(` title="link" href="%s"`, link.Title)))
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
				fontSize := ""
				switch heading.Level {
				case 1:
					fontSize = "text-5xl"
				case 2:
					fontSize = "text-4xl"
				case 3:
					fontSize = "text-3xl"
				case 4:
					fontSize = "text-2xl"
				case 5:
					fontSize = "text-xl"
				case 6:
					fontSize = "text-lg"
				}
				attr := heading.Attribute
				if attr == nil {
					attr = &ast.Attribute{}
				}
				attr.Classes = append(
					attr.Classes,
					[]byte(fmt.Sprintf(
						"%s my-4 font-bold leading-7 text-green-900",
						fontSize,
					)),
				)
				heading.Attribute = attr
			}
			if list, ok := node.(*ast.List); ok {
				listClass := "list-disc"
				if list.ListFlags&ast.ListTypeOrdered != 0 {
					listClass = "list-decimal"
				}
				if entering {
					// Start of the list
					fmt.Fprintf(w, `<ul class="%s">`, listClass)
				} else {
					// End of the list
					fmt.Fprintf(w, "</ul>")
				}
				return ast.GoToNext, true
			}
			if li, ok := node.(*ast.ListItem); ok {
				attr := li.Attribute
				if attr == nil {
					attr = &ast.Attribute{}
				}
				attr.Classes = append(attr.Classes, []byte("text-lg"))
				if entering {
					w.Write([]byte(`<li>`))
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

// assumes lists can be wrapped and we will still find parent lists, for exmaple
// ````
// <div><ul><li>list item</li></ul></div>
// ````
func calculateListDepth(node ast.Node) int {
	depth := 0
	parent := node.GetParent()
	for parent != nil {
		if _, ok := parent.(*ast.List); ok {
			depth++
		}
		parent = parent.GetParent()
	}
	return depth
}
