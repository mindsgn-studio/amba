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
