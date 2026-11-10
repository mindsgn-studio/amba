package readalong

import (
	"github.com/mindsgn-studio/amba/internal/epub"
	"github.com/mindsgn-studio/amba/internal/textutil"
)

// ReviewPageWords is the most words one page of a Review holds. It keeps a
// page readable in the reader's drawer without asking the listener to scroll
// through a whole chapter.
const ReviewPageWords = 120

// Review is an attached ebook laid out for inspecting how much of it the
// narration reads. It is built from the book's own text, not from the content
// the reader shows, so a book that fell back to the transcript can still be
// reviewed.
type Review struct {
	// Pages are the book's text, in reading order, cut into drawer pages.
	Pages []ReviewPage
	// Synced is the number of printed words matched to the narration.
	Synced int
	// Total is the number of printed words in the book.
	Total int
}

// ReviewPage is one page of the attached ebook.
type ReviewPage struct {
	// Chapter is the page's place in the book.
	Chapter int
	// Title is the chapter's title.
	Title string
	// Accepted reports whether the reader uses this chapter's text. A
	// rejected chapter is shown in the drawer for inspection, but the reader
	// shows its transcript instead, so none of its words are marked synced.
	Accepted bool
	// Paragraphs are the page's paragraphs, in reading order.
	Paragraphs []ReviewParagraph
}

// ReviewParagraph is a run of words on a page.
type ReviewParagraph struct {
	Words []ReviewWord
}

// ReviewWord is one printed word on a page.
type ReviewWord struct {
	Text string
	// Synced reports that the word has a place in the narration.
	Synced bool
	// Word is false for punctuation-only tokens, such as a lone dash. They
	// are printed but carry no narration, so the drawer leaves them neutral
	// rather than calling them unsynced.
	Word bool
}

// Percent returns the share of printed words that the narration reads, from 0
// to 1.
func (r *Review) Percent() float64 {
	if r == nil || r.Total == 0 {
		return 0
	}
	return float64(r.Synced) / float64(r.Total)
}

// PageForChapter returns the first page showing chapter, or 0 when the
// chapter has no page.
func (r *Review) PageForChapter(chapter int) int {
	if r == nil {
		return 0
	}
	for index, page := range r.Pages {
		if page.Chapter == chapter {
			return index
		}
	}
	return 0
}

// BuildReview lays an ebook out for inspection, marking each printed word that
// the alignment matched to the narration. A nil alignment still produces a
// review: every word is shown as unsynced, which is what the listener should
// see when the book could not be matched at all.
func BuildReview(chapters []epub.Chapter, alignment *Alignment) *Review {
	ebook := extractEbook(chapters)
	if len(ebook.chapters) == 0 {
		return nil
	}

	accepted := make([]bool, len(ebook.chapters))
	syncedSource := make([]bool, len(ebook.sourceWords))
	if alignment != nil {
		for _, chapter := range alignment.Chapters {
			if chapter.Index < 0 || chapter.Index >= len(accepted) {
				continue
			}
			accepted[chapter.Index] = chapter.Accepted
			if !chapter.Accepted {
				// A rejected chapter is not used by the reader, even if the
				// matcher found some words in it. Marking those words synced
				// would promise highlighting the reader never does.
				continue
			}
			for _, match := range chapter.Matches {
				if match.EbookWordIndex >= 0 && match.EbookWordIndex < len(syncedSource) {
					syncedSource[match.EbookWordIndex] = true
				}
			}
		}
	}

	syncedFlat := make([]bool, len(ebook.flat))
	for sourceIndex, ref := range ebook.sourceRefs {
		if sourceIndex < len(syncedSource) && syncedSource[sourceIndex] {
			syncedFlat[ref.Flat] = true
		}
	}

	review := &Review{}
	page := ReviewPage{}
	pageWords := 0
	flat := 0

	flush := func() {
		if len(page.Paragraphs) == 0 {
			return
		}
		review.Pages = append(review.Pages, page)
		page = ReviewPage{}
		pageWords = 0
	}

	for chapterIndex, chapter := range ebook.chapters {
		// A page belongs to one chapter: it keeps the page's title and its
		// accepted state meaningful, and lets the drawer jump to a chapter.
		flush()
		for _, paragraph := range chapter.Paragraphs {
			var current *ReviewParagraph
			for _, token := range paragraph.Tokens {
				isWord := textutil.IsWord(token.Text)
				if isWord {
					review.Total++
					if syncedFlat[flat] {
						review.Synced++
					}
				}

				// Start a new page at a word boundary, so punctuation stays
				// with the word it belongs to.
				if isWord && pageWords >= ReviewPageWords {
					flush()
					current = nil
				}

				if len(page.Paragraphs) == 0 {
					page.Chapter = chapterIndex
					page.Title = chapter.Title
					page.Accepted = accepted[chapterIndex]
				}
				if current == nil {
					page.Paragraphs = append(page.Paragraphs, ReviewParagraph{})
					current = &page.Paragraphs[len(page.Paragraphs)-1]
				}
				current.Words = append(current.Words, ReviewWord{
					Text:   token.Text,
					Synced: syncedFlat[flat],
					Word:   isWord,
				})
				if isWord {
					pageWords++
				}
				flat++
			}
			// A paragraph always starts a new run on the page, even when the
			// previous one ended just before a page break.
			current = nil
		}
	}
	flush()

	return review
}
