package epub

import (
	"bytes"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/net/html"

	"github.com/mindsgn-studio/amba/internal/textutil"
)

// parseChapter extracts a title and readable paragraphs from an XHTML
// document. Inline markup is flattened; block elements start new paragraphs.
// The printed text is kept as it appears, while words also carry a normalised
// form and offsets for matching.
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

	var chapterText strings.Builder
	for _, text := range paragraphs {
		if chapterText.Len() > 0 {
			chapterText.WriteString("\n\n")
		}
		chapterText.WriteString(text)
	}

	offset := 0
	for _, text := range paragraphs {
		paragraph := analyzeParagraph(text, len(chapter.Paragraphs))
		chapter.Paragraphs = append(chapter.Paragraphs, paragraph)
		for _, sentence := range paragraph.Sentences {
			sentence.Index = len(chapter.Sentences)
			sentence.Start += offset
			sentence.End += offset
			for index := range sentence.Words {
				sentence.Words[index].Start += offset
				sentence.Words[index].End += offset
			}
			chapter.Sentences = append(chapter.Sentences, sentence)
		}
		offset += len(text) + 2
	}

	chapter.Text = chapterText.String()
	chapter.Normalized = normalizeText(chapter.Text)
	if chapter.Title == "" {
		chapter.Title = documentTitle(root)
	}
	return chapter, nil
}

// analyzeParagraph records the paragraph's printed text, its normalised form,
// and the words and sentences inside it. Offsets are byte offsets into the
// paragraph text.
func analyzeParagraph(text string, index int) Paragraph {
	paragraph := Paragraph{
		Index:      index,
		Text:       text,
		Normalized: normalizeText(text),
	}
	for _, span := range wordSpans(text, 0) {
		paragraph.Words = append(paragraph.Words, span.Text)
	}
	paragraph.Sentences = splitSentences(text, len(paragraph.Words) > 0)
	return paragraph
}

type wordSpan struct {
	Text  string
	Start int
	End   int
}

// wordSpans finds whitespace-separated words and their byte offsets in text.
func wordSpans(text string, base int) []wordSpan {
	var spans []wordSpan
	start := -1
	for index := 0; index < len(text); {
		r, size := utf8.DecodeRuneInString(text[index:])
		if unicode.IsSpace(r) {
			if start >= 0 {
				spans = append(spans, wordSpan{Text: text[start:index], Start: base + start, End: base + index})
				start = -1
			}
		} else if start < 0 {
			start = index
		}
		index += size
	}
	if start >= 0 {
		spans = append(spans, wordSpan{Text: text[start:], Start: base + start, End: base + len(text)})
	}
	return spans
}

// splitSentences cuts text after sentence-ending punctuation followed by
// whitespace or the end of the paragraph. It does not try to understand
// abbreviations; sentence boundaries are metadata only, and the reader never
// depends on them for highlighting.
func splitSentences(text string, withWords bool) []Sentence {
	if strings.TrimSpace(text) == "" {
		return nil
	}

	var sentences []Sentence
	start := 0
	index := 0
	for index < len(text) {
		r, size := utf8.DecodeRuneInString(text[index:])
		index += size
		if r != '.' && r != '!' && r != '?' && r != '\u2026' {
			continue
		}

		// Include closing quotes and brackets in the sentence.
		for index < len(text) {
			next, nextSize := utf8.DecodeRuneInString(text[index:])
			if next == '"' || next == '\'' || next == '\u201d' || next == '\u2019' ||
				next == ')' || next == ']' || next == '\u00bb' {
				index += nextSize
				continue
			}
			break
		}
		if index < len(text) {
			next, _ := utf8.DecodeRuneInString(text[index:])
			if !unicode.IsSpace(next) {
				continue
			}
		}

		if sentence := makeSentence(text, start, index, withWords); sentence != nil {
			sentences = append(sentences, *sentence)
		}
		for index < len(text) {
			next, nextSize := utf8.DecodeRuneInString(text[index:])
			if !unicode.IsSpace(next) {
				break
			}
			index += nextSize
		}
		start = index
	}

	if start < len(text) {
		if sentence := makeSentence(text, start, len(text), withWords); sentence != nil {
			sentences = append(sentences, *sentence)
		}
	}
	return sentences
}

func makeSentence(text string, start, end int, withWords bool) *Sentence {
	for start < end {
		r, size := utf8.DecodeRuneInString(text[start:end])
		if !unicode.IsSpace(r) {
			break
		}
		start += size
	}
	for end > start {
		r, size := utf8.DecodeLastRuneInString(text[start:end])
		if !unicode.IsSpace(r) {
			break
		}
		end -= size
	}
	if start >= end {
		return nil
	}

	value := text[start:end]
	sentence := &Sentence{
		Text:       value,
		Normalized: normalizeText(value),
		Start:      start,
		End:        end,
	}
	if withWords {
		for _, span := range wordSpans(value, start) {
			sentence.Words = append(sentence.Words, Word{
				Text:       span.Text,
				Normalized: textutil.Normalize(span.Text),
				Start:      span.Start,
				End:        span.End,
			})
		}
	}
	return sentence
}

func normalizeText(text string) string {
	words := strings.Fields(text)
	for index, word := range words {
		words[index] = textutil.Normalize(word)
	}
	return strings.Join(words, " ")
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
