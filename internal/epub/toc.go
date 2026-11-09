package epub

import (
	"bytes"
	"encoding/xml"
	"path"
	"strings"

	"golang.org/x/net/html"
)

// TableOfContents returns the book's table of contents in reading order.
// EPUB 3 navigation documents and EPUB 2 NCX files are both supported.
func (d *Document) TableOfContents() ([]TOCEntry, error) {
	d.tocOnce.Do(func() {
		d.toc, d.tocErr = d.loadTOC()
	})
	return d.toc, d.tocErr
}

func (d *Document) tocTitle(href string) string {
	entries, err := d.TableOfContents()
	if err != nil {
		return ""
	}
	wanted := path.Clean(href)
	for _, entry := range entries {
		if path.Clean(entry.Href) == wanted && entry.Title != "" {
			return entry.Title
		}
	}
	return ""
}

func (d *Document) loadTOC() ([]TOCEntry, error) {
	if item, ok := d.navItem(); ok {
		if entries, err := d.loadNavTOC(item); err == nil && len(entries) > 0 {
			return entries, nil
		}
	}
	if item, ok := d.ncxItem(); ok {
		return d.loadNCXTOC(item)
	}
	return nil, nil
}

func (d *Document) navItem() (manifestItem, bool) {
	for _, id := range d.manifestOrder {
		item, ok := d.manifest[id]
		if ok && hasProperty(item.Properties, "nav") {
			return item, true
		}
	}
	return manifestItem{}, false
}

func (d *Document) ncxItem() (manifestItem, bool) {
	if d.tocID != "" {
		if item, ok := d.manifest[d.tocID]; ok {
			return item, true
		}
	}
	for _, id := range d.manifestOrder {
		item, ok := d.manifest[id]
		if ok && (strings.EqualFold(strings.TrimSpace(item.MediaType), "application/x-dtbncx+xml") ||
			strings.EqualFold(path.Ext(item.Href), ".ncx")) {
			return item, true
		}
	}
	return manifestItem{}, false
}

func (d *Document) loadNavTOC(item manifestItem) ([]TOCEntry, error) {
	documentPath := resolveHref(d.opfDir, item.Href)
	data, err := d.read(documentPath)
	if err != nil {
		return nil, err
	}
	root, err := html.Parse(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	base := path.Dir(documentPath)
	if base == "." {
		base = ""
	}

	nav := tocNavigation(root)
	if nav == nil {
		return nil, nil
	}

	var entries []TOCEntry
	var walk func(*html.Node, int)
	walk = func(node *html.Node, level int) {
		if node.Type == html.ElementNode {
			switch node.Data {
			case "ol", "ul":
				level++
			case "a":
				href := ""
				for _, attr := range node.Attr {
					if attr.Key == "href" {
						href = attr.Val
						break
					}
				}
				title := strings.Join(strings.Fields(textOf(node)), " ")
				if href != "" && title != "" {
					entries = append(entries, TOCEntry{
						Title: title,
						Href:  resolveHref(base, href),
						Level: level,
					})
				}
				return
			case "nav":
				// Do not descend into nested navigation regions such as
				// landmarks or a page list.
				if node != nav {
					return
				}
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child, level)
		}
	}
	walk(nav, 0)
	return entries, nil
}

// tocNavigation finds the EPUB 3 table-of-contents nav. If the document has
// no marked nav, the first nav (or the document itself) is used.
func tocNavigation(root *html.Node) *html.Node {
	var first *html.Node
	var found *html.Node
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == "nav" {
			if first == nil {
				first = node
			}
			if found == nil && isTOCNav(node) {
				found = node
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(root)
	if found != nil {
		return found
	}
	if first != nil {
		return first
	}
	return findElement(root, "body")
}

func isTOCNav(node *html.Node) bool {
	for _, attr := range node.Attr {
		key := strings.ToLower(attr.Key)
		value := strings.ToLower(attr.Val)
		switch {
		case key == "role" && strings.Contains(value, "doc-toc"):
			return true
		case (key == "type" || key == "epub:type") && strings.Contains(value, "toc"):
			return true
		}
	}
	return false
}

type ncxDocument struct {
	NavMap struct {
		Points []ncxPoint `xml:"navPoint"`
	} `xml:"navMap"`
}

type ncxPoint struct {
	Label struct {
		Text string `xml:"text"`
	} `xml:"navLabel"`
	Content struct {
		Src string `xml:"src,attr"`
	} `xml:"content"`
	Points []ncxPoint `xml:"navPoint"`
}

func (d *Document) loadNCXTOC(item manifestItem) ([]TOCEntry, error) {
	documentPath := resolveHref(d.opfDir, item.Href)
	data, err := d.read(documentPath)
	if err != nil {
		return nil, err
	}
	var ncx ncxDocument
	if err := xml.Unmarshal(data, &ncx); err != nil {
		return nil, err
	}
	base := path.Dir(documentPath)
	if base == "." {
		base = ""
	}

	var entries []TOCEntry
	var walk func([]ncxPoint, int)
	walk = func(points []ncxPoint, level int) {
		for _, point := range points {
			title := strings.Join(strings.Fields(point.Label.Text), " ")
			if title != "" && point.Content.Src != "" {
				entries = append(entries, TOCEntry{
					Title: title,
					Href:  resolveHref(base, point.Content.Src),
					Level: level,
				})
			}
			walk(point.Points, level+1)
		}
	}
	walk(ncx.NavMap.Points, 0)
	return entries, nil
}
