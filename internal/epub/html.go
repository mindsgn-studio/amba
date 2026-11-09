package epub

import (
	"bytes"
	"strings"

	"golang.org/x/net/html"
)

// parseChapter extracts a title and readable paragraphs from an XHTML
// document. Inline markup is flattened; block elements start new paragraphs.
func parseChapter(data []byte) (Chapter, error) {
	root, err := html.Parse(bytes.NewReader(data))
	if err != nil {
		return Chapter{}, err
	}

	chapter := Chapter{Title: firstHeading(root)}
	body := findElement(root, "body")
	if body == nil {
		body = root
	}

	var paragraphs []string
	var current strings.Builder

	flush := func() {
		text := strings.Join(strings.Fields(current.String()), " ")
		current.Reset()
		if text != "" {
			paragraphs = append(paragraphs, text)
		}
	}

	var walk func(*html.Node)
	walk = func(node *html.Node) {
		switch node.Type {
		case html.TextNode:
			current.WriteString(node.Data)
			return
		case html.ElementNode:
			switch node.Data {
			case "script", "style", "head", "nav", "svg", "noscript":
				return
			case "h1", "h2", "h3", "h4", "h5", "h6":
				// The heading is the chapter's title, shown by the reader's
				// header. It must not also become the first paragraph.
				flush()
				return
			case "br":
				flush()
				return
			}
			if isBlock(node.Data) {
				flush()
				for child := node.FirstChild; child != nil; child = child.NextSibling {
					walk(child)
				}
				flush()
				return
			}
		}

		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}

	walk(body)
	flush()

	for _, text := range paragraphs {
		words := strings.Fields(text)
		if len(words) == 0 {
			continue
		}
		chapter.Paragraphs = append(chapter.Paragraphs, Paragraph{Words: words})
	}
	if chapter.Title == "" {
		chapter.Title = documentTitle(root)
	}
	return chapter, nil
}

func isBlock(tag string) bool {
	switch tag {
	case "address", "article", "aside", "blockquote", "dd", "div", "dl", "dt",
		"figcaption", "figure", "footer", "form", "h1", "h2", "h3", "h4", "h5",
		"h6", "header", "hr", "li", "main", "ol", "p", "pre", "section", "table",
		"tbody", "td", "tfoot", "th", "thead", "tr", "ul":
		return true
	}
	return false
}

func findElement(node *html.Node, tag string) *html.Node {
	if node.Type == html.ElementNode && node.Data == tag {
		return node
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if found := findElement(child, tag); found != nil {
			return found
		}
	}
	return nil
}

func documentTitle(root *html.Node) string {
	if title := findElement(root, "title"); title != nil {
		return strings.Join(strings.Fields(textOf(title)), " ")
	}
	return ""
}

func firstHeading(root *html.Node) string {
	for _, tag := range []string{"h1", "h2", "h3", "h4", "h5", "h6"} {
		if heading := findElement(root, tag); heading != nil {
			if text := strings.Join(strings.Fields(textOf(heading)), " "); text != "" {
				return text
			}
		}
	}
	return ""
}

func textOf(node *html.Node) string {
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
			b.WriteByte(' ')
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(node)
	return b.String()
}
