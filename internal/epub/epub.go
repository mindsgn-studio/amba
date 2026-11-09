// Package epub reads the parts of an EPUB that a read-along needs: its
// metadata, cover, reading order and plain text.
package epub

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"strings"
	"sync"
)

// Metadata is the information a library shows without opening the whole book.
type Metadata struct {
	Title          string
	Author         string
	Language       string
	Cover          []byte
	CoverMediaType string
}

// Chapter is one document of the reading order.
type Chapter struct {
	Title      string
	Paragraphs []Paragraph
}

// Paragraph is a run of words as printed, with punctuation kept.
type Paragraph struct {
	Words []string
}

// Document is an opened EPUB. Its chapters are parsed on first use.
type Document struct {
	Path     string
	Metadata Metadata

	files    *zip.Reader
	opfDir   string
	manifest map[string]manifestItem
	spine    []spineItem

	once        sync.Once
	chapters    []Chapter
	chaptersErr error
}

type manifestItem struct {
	ID         string
	Href       string
	MediaType  string
	Properties string
}

type spineItem struct {
	IDRef  string
	Linear string
}

// Open reads an EPUB's container, package document, metadata and cover.
func Open(filename string) (*Document, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("read EPUB archive: %w", err)
	}

	document := &Document{
		Path:     filename,
		files:    reader,
		manifest: make(map[string]manifestItem),
	}

	containerPath := "META-INF/container.xml"
	containerData, err := document.read(containerPath)
	if err != nil {
		return nil, fmt.Errorf("read EPUB container: %w", err)
	}

	var container containerXML
	if err := xml.Unmarshal(containerData, &container); err != nil {
		return nil, fmt.Errorf("parse EPUB container: %w", err)
	}

	opfPath := ""
	for _, rootfile := range container.Rootfiles {
		if strings.TrimSpace(rootfile.FullPath) != "" {
			opfPath = rootfile.FullPath
			break
		}
	}
	if opfPath == "" {
		// Some malformed EPUBs omit container.xml's rootfile. Fall back to the
		// first package document in the archive.
		for _, file := range reader.File {
			if strings.EqualFold(path.Ext(file.Name), ".opf") {
				opfPath = file.Name
				break
			}
		}
	}
	if opfPath == "" {
		return nil, errors.New("EPUB has no package document")
	}

	document.opfDir = path.Dir(opfPath)
	if document.opfDir == "." {
		document.opfDir = ""
	}

	opfData, err := document.read(opfPath)
	if err != nil {
		return nil, fmt.Errorf("read EPUB package: %w", err)
	}

	var pkg packageXML
	if err := xml.Unmarshal(opfData, &pkg); err != nil {
		return nil, fmt.Errorf("parse EPUB package: %w", err)
	}

	document.Metadata = Metadata{
		Title:    strings.TrimSpace(pkg.Metadata.Title),
		Author:   strings.TrimSpace(pkg.Metadata.Creator),
		Language: strings.TrimSpace(pkg.Metadata.Language),
	}

	for _, item := range pkg.Manifest.Items {
		document.manifest[item.ID] = manifestItem{
			ID:         item.ID,
			Href:       item.Href,
			MediaType:  item.MediaType,
			Properties: item.Properties,
		}
	}
	for _, item := range pkg.Spine.ItemRefs {
		document.spine = append(document.spine, spineItem{IDRef: item.IDRef, Linear: item.Linear})
	}

	document.loadCover(pkg)

	return document, nil
}

// MetadataOf reads only the metadata and cover of an EPUB.
func MetadataOf(filename string) (Metadata, error) {
	document, err := Open(filename)
	if err != nil {
		return Metadata{}, err
	}
	return document.Metadata, nil
}

// Chapters returns the documents of the reading order that contain text.
func (d *Document) Chapters() ([]Chapter, error) {
	d.once.Do(func() {
		for _, itemRef := range d.spine {
			if strings.EqualFold(itemRef.Linear, "no") {
				continue
			}
			item, ok := d.manifest[itemRef.IDRef]
			if !ok {
				continue
			}
			if !isHTML(item.MediaType, item.Href) {
				continue
			}
			data, err := d.read(resolveHref(d.opfDir, item.Href))
			if err != nil {
				continue
			}
			chapter, err := parseChapter(data)
			if err != nil {
				continue
			}
			if len(chapter.Paragraphs) == 0 {
				continue
			}
			if chapter.Title == "" {
				chapter.Title = fmt.Sprintf("Chapter %d", len(d.chapters)+1)
			}
			d.chapters = append(d.chapters, chapter)
		}
		if len(d.chapters) == 0 {
			d.chaptersErr = errors.New("EPUB has no readable chapters")
		}
	})
	return d.chapters, d.chaptersErr
}

func (d *Document) loadCover(pkg packageXML) {
	coverID := ""
	for _, meta := range pkg.Metadata.Metas {
		if strings.EqualFold(meta.Name, "cover") {
			coverID = strings.TrimSpace(meta.Content)
			break
		}
	}

	var cover manifestItem
	found := false
	if coverID != "" {
		cover, found = d.manifest[coverID]
	}
	if !found {
		for _, item := range pkg.Manifest.Items {
			if hasProperty(item.Properties, "cover-image") {
				cover = manifestItem{
					ID:         item.ID,
					Href:       item.Href,
					MediaType:  item.MediaType,
					Properties: item.Properties,
				}
				found = true
				break
			}
		}
	}
	if !found {
		return
	}

	data, err := d.read(resolveHref(d.opfDir, cover.Href))
	if err != nil || len(data) == 0 {
		return
	}
	d.Metadata.Cover = data
	d.Metadata.CoverMediaType = cover.MediaType
}

func (d *Document) read(name string) ([]byte, error) {
	name = path.Clean(strings.TrimPrefix(name, "/"))
	for _, file := range d.files.File {
		if file.Name == name {
			return readZipFile(file)
		}
	}
	// ZIP entries are case-sensitive, but some EPUBs disagree with their own
	// package document about case. Try once more without it.
	for _, file := range d.files.File {
		if strings.EqualFold(file.Name, name) {
			return readZipFile(file)
		}
	}
	return nil, fmt.Errorf("EPUB entry not found: %s", name)
}

func readZipFile(file *zip.File) ([]byte, error) {
	reader, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(reader)
}

func resolveHref(base, href string) string {
	href = strings.TrimSpace(href)
	if unescaped, err := url.PathUnescape(href); err == nil {
		href = unescaped
	}
	if hash := strings.IndexByte(href, '#'); hash >= 0 {
		href = href[:hash]
	}
	href = strings.TrimPrefix(href, "/")
	if base == "" {
		return path.Clean(href)
	}
	return path.Clean(path.Join(base, href))
}

func isHTML(mediaType, href string) bool {
	mediaType = strings.ToLower(strings.TrimSpace(mediaType))
	if strings.Contains(mediaType, "xhtml") || strings.Contains(mediaType, "html") {
		return true
	}
	switch strings.ToLower(path.Ext(href)) {
	case ".xhtml", ".html", ".htm":
		return true
	}
	return false
}

func hasProperty(properties, wanted string) bool {
	for _, property := range strings.Fields(properties) {
		if strings.EqualFold(property, wanted) {
			return true
		}
	}
	return false
}

type containerXML struct {
	Rootfiles []struct {
		FullPath string `xml:"full-path,attr"`
	} `xml:"rootfiles>rootfile"`
}

type packageXML struct {
	Metadata struct {
		Title    string `xml:"title"`
		Creator  string `xml:"creator"`
		Language string `xml:"language"`
		Metas    []struct {
			Name     string `xml:"name,attr"`
			Content  string `xml:"content,attr"`
			Property string `xml:"property,attr"`
		} `xml:"meta"`
	} `xml:"metadata"`
	Manifest struct {
		Items []struct {
			ID         string `xml:"id,attr"`
			Href       string `xml:"href,attr"`
			MediaType  string `xml:"media-type,attr"`
			Properties string `xml:"properties,attr"`
		} `xml:"item"`
	} `xml:"manifest"`
	Spine struct {
		ItemRefs []struct {
			IDRef  string `xml:"idref,attr"`
			Linear string `xml:"linear,attr"`
		} `xml:"itemref"`
	} `xml:"spine"`
}
