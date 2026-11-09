package epub

import (
	"archive/zip"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

type entry struct {
	name string
	data []byte
}

func writeEPUB(t *testing.T, entries ...entry) string {
	t.Helper()

	filename := filepath.Join(t.TempDir(), "book.epub")
	file, err := os.Create(filename)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	writer := zip.NewWriter(file)
	for _, item := range entries {
		part, err := writer.Create(item.name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := part.Write(item.data); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return filename
}

func testEPUB(t *testing.T) string {
	t.Helper()

	container := []byte(`<?xml version="1.0"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>`)

	pack := []byte(`<?xml version="1.0" encoding="utf-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="3.0">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>Test Book</dc:title>
    <dc:creator>A. Writer</dc:creator>
    <dc:language>en</dc:language>
    <meta name="cover" content="cover-image"/>
  </metadata>
  <manifest>
    <item id="nav" href="nav.xhtml" media-type="application/xhtml+xml" properties="nav"/>
    <item id="ch1" href="text/chapter1.xhtml" media-type="application/xhtml+xml"/>
    <item id="ch2" href="text/chapter2.xhtml" media-type="application/xhtml+xml"/>
    <item id="cover-image" href="images/cover.png" media-type="image/png" properties="cover-image"/>
  </manifest>
  <spine>
    <itemref idref="nav" linear="no"/>
    <itemref idref="ch1"/>
    <itemref idref="ch2"/>
  </spine>
</package>`)

	chapter1 := []byte(`<html xmlns="http://www.w3.org/1999/xhtml">
  <head><title>Title Tag</title></head>
  <body>
    <h1>Chapter One</h1>
    <p>Hello <em>world</em>.</p>
    <p>Second &amp; last.</p>
  </body>
</html>`)

	chapter2 := []byte(`<html><body><div><p>Only text</p></div></body></html>`)

	return writeEPUB(t,
		entry{"mimetype", []byte("application/epub+zip")},
		entry{"META-INF/container.xml", container},
		entry{"OEBPS/content.opf", pack},
		entry{"OEBPS/nav.xhtml", []byte(`<html><body><nav>Contents</nav></body></html>`)},
		entry{"OEBPS/text/chapter1.xhtml", chapter1},
		entry{"OEBPS/text/chapter2.xhtml", chapter2},
		entry{"OEBPS/images/cover.png", []byte("PNGDATA")},
	)
}

func TestOpenReadsMetadataAndCover(t *testing.T) {
	document, err := Open(testEPUB(t))
	if err != nil {
		t.Fatal(err)
	}

	metadata := document.Metadata
	if metadata.Title != "Test Book" {
		t.Errorf("Title = %q, want %q", metadata.Title, "Test Book")
	}
	if metadata.Author != "A. Writer" {
		t.Errorf("Author = %q, want %q", metadata.Author, "A. Writer")
	}
	if metadata.Language != "en" {
		t.Errorf("Language = %q, want %q", metadata.Language, "en")
	}
	if string(metadata.Cover) != "PNGDATA" {
		t.Errorf("Cover = %q, want %q", metadata.Cover, "PNGDATA")
	}
	if metadata.CoverMediaType != "image/png" {
		t.Errorf("CoverMediaType = %q, want %q", metadata.CoverMediaType, "image/png")
	}
}

func TestMetadataOf(t *testing.T) {
	metadata, err := MetadataOf(testEPUB(t))
	if err != nil {
		t.Fatal(err)
	}
	if metadata.Title != "Test Book" || metadata.Author != "A. Writer" {
		t.Fatalf("MetadataOf() = %+v, want the package metadata", metadata)
	}
}

func TestChapters(t *testing.T) {
	document, err := Open(testEPUB(t))
	if err != nil {
		t.Fatal(err)
	}

	chapters, err := document.Chapters()
	if err != nil {
		t.Fatal(err)
	}
	if len(chapters) != 2 {
		t.Fatalf("Chapters() returned %d chapters, want 2", len(chapters))
	}

	first := chapters[0]
	if first.Title != "Chapter One" {
		t.Errorf("first Title = %q, want %q", first.Title, "Chapter One")
	}
	wantParagraphs := [][]string{
		{"Hello", "world."},
		{"Second", "&", "last."},
	}
	if len(first.Paragraphs) != len(wantParagraphs) {
		t.Fatalf("first chapter has %d paragraphs, want %d", len(first.Paragraphs), len(wantParagraphs))
	}
	for index, want := range wantParagraphs {
		if !reflect.DeepEqual(first.Paragraphs[index].Words, want) {
			t.Errorf("paragraph %d = %q, want %q", index, first.Paragraphs[index].Words, want)
		}
	}

	second := chapters[1]
	if second.Title != "Chapter 2" {
		t.Errorf("second Title = %q, want %q", second.Title, "Chapter 2")
	}
	if len(second.Paragraphs) != 1 || !reflect.DeepEqual(second.Paragraphs[0].Words, []string{"Only", "text"}) {
		t.Errorf("second chapter = %+v, want the single paragraph", second.Paragraphs)
	}

	// The chapters are parsed once and reused.
	again, err := document.Chapters()
	if err != nil {
		t.Fatal(err)
	}
	if &again[0] != &chapters[0] {
		t.Error("Chapters() parsed the book again instead of reusing its result")
	}
}

func TestOpenRejectsArchiveWithoutContainer(t *testing.T) {
	filename := writeEPUB(t, entry{"random.txt", []byte("nothing")})
	if _, err := Open(filename); err == nil {
		t.Fatal("Open() accepted an archive with no container")
	}
}

func TestOpenRejectsMissingFile(t *testing.T) {
	if _, err := Open(filepath.Join(t.TempDir(), "missing.epub")); err == nil {
		t.Fatal("Open() accepted a missing file")
	}
}

func TestResolveHref(t *testing.T) {
	tests := []struct {
		base string
		href string
		want string
	}{
		{"OEBPS", "text/chapter1.xhtml", "OEBPS/text/chapter1.xhtml"},
		{"OEBPS", "text/ch%201.xhtml#frag", "OEBPS/text/ch 1.xhtml"},
		{"OEBPS", "/root.xhtml", "OEBPS/root.xhtml"},
		{"OEBPS", "../shared/style.css", "shared/style.css"},
		{"", "chapter.xhtml", "chapter.xhtml"},
	}
	for _, test := range tests {
		if got := resolveHref(test.base, test.href); got != test.want {
			t.Errorf("resolveHref(%q, %q) = %q, want %q", test.base, test.href, got, test.want)
		}
	}
}

func TestChaptersPreserveTextSentencesAndOffsets(t *testing.T) {
	document, err := Open(testEPUB(t))
	if err != nil {
		t.Fatal(err)
	}
	chapters, err := document.Chapters()
	if err != nil {
		t.Fatal(err)
	}
	first := chapters[0]

	if first.Text != "Hello world.\n\nSecond & last." {
		t.Errorf("chapter text = %q, want the printed paragraphs", first.Text)
	}
	if first.Normalized != "hello world second and last" {
		t.Errorf("chapter normalized text = %q", first.Normalized)
	}
	if len(first.Sentences) != 2 {
		t.Fatalf("chapter has %d sentences, want 2", len(first.Sentences))
	}

	paragraph := first.Paragraphs[0]
	if paragraph.Text != "Hello world." || paragraph.Normalized != "hello world" {
		t.Errorf("paragraph = %+v, want printed and normalized text", paragraph)
	}
	if len(paragraph.Sentences) != 1 {
		t.Fatalf("paragraph has %d sentences, want 1", len(paragraph.Sentences))
	}
	sentence := paragraph.Sentences[0]
	if sentence.Text != "Hello world." || sentence.Start != 0 || sentence.End != len("Hello world.") {
		t.Errorf("sentence = %+v, want the whole paragraph at its offsets", sentence)
	}
	if len(sentence.Words) != 2 {
		t.Fatalf("sentence has %d words, want 2", len(sentence.Words))
	}
	for _, word := range sentence.Words {
		if got := paragraph.Text[word.Start:word.End]; got != word.Text {
			t.Errorf("word %+v offsets select %q", word, got)
		}
	}
	if sentence.Words[1].Normalized != "world" {
		t.Errorf("normalized word = %q, want world", sentence.Words[1].Normalized)
	}
}

func TestTableOfContentsFromNavigationDocument(t *testing.T) {
	filename := writeEPUB(t,
		entry{"META-INF/container.xml", []byte(`<?xml version="1.0"?>
<container xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles><rootfile full-path="OEBPS/content.opf"/></rootfiles>
</container>`)},
		entry{"OEBPS/content.opf", []byte(`<?xml version="1.0"?>
<package xmlns="http://www.idpf.org/2007/opf">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/"><dc:title>Contents Test</dc:title></metadata>
  <manifest>
    <item id="nav" href="nav.xhtml" media-type="application/xhtml+xml" properties="nav"/>
    <item id="ch1" href="text/one.xhtml" media-type="application/xhtml+xml"/>
    <item id="ch2" href="text/two.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine><itemref idref="ch1"/><itemref idref="ch2"/></spine>
</package>`)},
		entry{"OEBPS/nav.xhtml", []byte(`<html><body>
<nav epub:type="landmarks"><ol><li><a href="text/one.xhtml">Start</a></li></ol></nav>
<nav epub:type="toc"><ol>
  <li><a href="text/one.xhtml">First Part</a></li>
  <li><a href="text/two.xhtml#start">Second Part</a></li>
</ol></nav></body></html>`)},
		entry{"OEBPS/text/one.xhtml", []byte(`<html><body><p>One.</p></body></html>`)},
		entry{"OEBPS/text/two.xhtml", []byte(`<html><body><p>Two.</p></body></html>`)},
	)

	document, err := Open(filename)
	if err != nil {
		t.Fatal(err)
	}
	entries, err := document.TableOfContents()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("TableOfContents() = %+v, want 2 entries", entries)
	}
	if entries[0].Title != "First Part" || entries[0].Href != "OEBPS/text/one.xhtml" {
		t.Errorf("first entry = %+v", entries[0])
	}
	if entries[1].Href != "OEBPS/text/two.xhtml" {
		t.Errorf("second entry = %+v, want the fragment removed", entries[1])
	}

	chapters, err := document.Chapters()
	if err != nil {
		t.Fatal(err)
	}
	if chapters[0].Title != "First Part" || chapters[1].Title != "Second Part" {
		t.Errorf("chapter titles = %q, %q, want the table of contents", chapters[0].Title, chapters[1].Title)
	}
}

func TestOpenRejectsInvalidArchive(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "broken.epub")
	if err := os.WriteFile(filename, []byte("this is not a zip archive"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(filename); err == nil {
		t.Fatal("Open() accepted a non-ZIP file")
	}
}

func TestTableOfContentsFromNCX(t *testing.T) {
	filename := writeEPUB(t,
		entry{"META-INF/container.xml", []byte(`<?xml version="1.0"?>
<container xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles><rootfile full-path="OEBPS/content.opf"/></rootfiles>
</container>`)},
		entry{"OEBPS/content.opf", []byte(`<?xml version="1.0"?>
<package xmlns="http://www.idpf.org/2007/opf">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/"><dc:title>NCX Test</dc:title></metadata>
  <manifest>
    <item id="ncx" href="toc.ncx" media-type="application/x-dtbncx+xml"/>
    <item id="ch1" href="text/one.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine toc="ncx"><itemref idref="ch1"/></spine>
</package>`)},
		entry{"OEBPS/toc.ncx", []byte(`<?xml version="1.0"?>
<ncx xmlns="http://www.daisy.org/z3986/2005/ncx/">
  <navMap>
    <navPoint>
      <navLabel><text>Opening</text></navLabel>
      <content src="text/one.xhtml"/>
      <navPoint>
        <navLabel><text>Subsection</text></navLabel>
        <content src="text/one.xhtml#sub"/>
      </navPoint>
    </navPoint>
  </navMap>
</ncx>`)},
		entry{"OEBPS/text/one.xhtml", []byte(`<html><body><p>One.</p></body></html>`)},
	)

	document, err := Open(filename)
	if err != nil {
		t.Fatal(err)
	}
	entries, err := document.TableOfContents()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("TableOfContents() = %+v, want 2 entries", entries)
	}
	if entries[0].Title != "Opening" || entries[0].Level != 0 {
		t.Errorf("first entry = %+v", entries[0])
	}
	if entries[1].Title != "Subsection" || entries[1].Level != 1 {
		t.Errorf("nested entry = %+v", entries[1])
	}
	if entries[1].Href != "OEBPS/text/one.xhtml" {
		t.Errorf("nested href = %q, want the fragment removed", entries[1].Href)
	}

	chapters, err := document.Chapters()
	if err != nil {
		t.Fatal(err)
	}
	if chapters[0].Title != "Opening" {
		t.Errorf("chapter title = %q, want the NCX label", chapters[0].Title)
	}
}
