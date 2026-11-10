package readalong

import (
	"fmt"
	"testing"

	"github.com/mindsgn-studio/amba/internal/align"
	"github.com/mindsgn-studio/amba/internal/epub"
)

func reviewChapter(title string, paragraphs ...[]string) epub.Chapter {
	chapter := epub.Chapter{Title: title}
	for _, words := range paragraphs {
		chapter.Paragraphs = append(chapter.Paragraphs, epub.Paragraph{Words: words})
	}
	return chapter
}

func TestBuildReviewMarksMatchedWords(t *testing.T) {
	chapters := []epub.Chapter{
		reviewChapter("One", []string{"The", "cat", "sat."}, []string{"Then", "it", "slept."}),
		reviewChapter("Two", []string{"A", "dog."}),
	}
	alignment := &Alignment{Chapters: []ChapterAlignment{
		{
			Index: 0,
			AlignmentResult: align.AlignmentResult{
				Accepted: true,
				Matches: []align.WordMatch{
					{EbookWordIndex: 0, AudioWordIndex: 0},
					{EbookWordIndex: 1, AudioWordIndex: 1},
					{EbookWordIndex: 2, AudioWordIndex: 2},
				},
			},
		},
		{
			Index: 1,
			AlignmentResult: align.AlignmentResult{
				Accepted: false,
				Matches: []align.WordMatch{
					{EbookWordIndex: 6, AudioWordIndex: 3},
					{EbookWordIndex: 7, AudioWordIndex: 4},
				},
			},
		},
	}}

	review := BuildReview(chapters, alignment)
	if review == nil {
		t.Fatal("BuildReview() = nil")
	}
	if review.Total != 8 {
		t.Errorf("Total = %d, want 8", review.Total)
	}
	if review.Synced != 3 {
		t.Errorf("Synced = %d, want 3 (a rejected chapter is never synced)", review.Synced)
	}
	if len(review.Pages) != 2 {
		t.Fatalf("pages = %d, want one per chapter", len(review.Pages))
	}

	first := review.Pages[0]
	if first.Chapter != 0 || first.Title != "One" || !first.Accepted {
		t.Errorf("first page = %+v, want accepted chapter One", first)
	}
	if got := first.Paragraphs[0].Words; got[0].Synced != true || got[1].Synced != true || got[2].Synced != true {
		t.Errorf("first paragraph = %+v, want every word synced", got)
	}
	if got := first.Paragraphs[1].Words; got[0].Synced || got[1].Synced || got[2].Synced {
		t.Errorf("second paragraph = %+v, want every word unsynced", got)
	}

	second := review.Pages[1]
	if second.Chapter != 1 || second.Title != "Two" || second.Accepted {
		t.Errorf("second page = %+v, want rejected chapter Two", second)
	}
	for _, word := range second.Paragraphs[0].Words {
		if word.Synced {
			t.Errorf("rejected chapter word %q is marked synced", word.Text)
		}
	}
}

func TestBuildReviewPaginatesAtWordBoundaries(t *testing.T) {
	words := make([]string, 0, ReviewPageWords+ReviewPageWords+10)
	for index := 0; index < cap(words); index++ {
		words = append(words, fmt.Sprintf("word%d", index))
	}
	chapters := []epub.Chapter{reviewChapter("Long", words)}

	review := BuildReview(chapters, nil)
	if review == nil {
		t.Fatal("BuildReview() = nil")
	}
	if review.Total != len(words) {
		t.Errorf("Total = %d, want %d", review.Total, len(words))
	}
	if review.Synced != 0 {
		t.Errorf("Synced = %d, want none without an alignment", review.Synced)
	}
	if len(review.Pages) != 3 {
		t.Fatalf("pages = %d, want 3", len(review.Pages))
	}

	total := 0
	for index, page := range review.Pages {
		count := 0
		for _, paragraph := range page.Paragraphs {
			count += len(paragraph.Words)
		}
		if count > ReviewPageWords {
			t.Errorf("page %d holds %d words, want at most %d", index, count, ReviewPageWords)
		}
		total += count
	}
	if total != len(words) {
		t.Errorf("pages hold %d words, want %d", total, len(words))
	}
}

func TestBuildReviewKeepsPunctuationNeutral(t *testing.T) {
	chapters := []epub.Chapter{reviewChapter("Punctuation", []string{"Hello", "—", "world"})}
	alignment := &Alignment{Chapters: []ChapterAlignment{{
		Index: 0,
		AlignmentResult: align.AlignmentResult{
			Accepted: true,
			Matches:  []align.WordMatch{{EbookWordIndex: 0, AudioWordIndex: 0}},
		},
	}}}

	review := BuildReview(chapters, alignment)
	if review == nil {
		t.Fatal("BuildReview() = nil")
	}
	words := review.Pages[0].Paragraphs[0].Words
	if !words[0].Synced || !words[0].Word {
		t.Errorf("hello = %+v, want a synced word", words[0])
	}
	if words[1].Synced || words[1].Word {
		t.Errorf("dash = %+v, want a neutral punctuation token", words[1])
	}
	if words[2].Synced || !words[2].Word {
		t.Errorf("world = %+v, want an unsynced word", words[2])
	}
	if review.Total != 2 {
		t.Errorf("Total = %d, want only the two words", review.Total)
	}
}

func TestReviewPageForChapter(t *testing.T) {
	first := make([]string, ReviewPageWords+5)
	for index := range first {
		first[index] = fmt.Sprintf("word%d", index)
	}
	chapters := []epub.Chapter{
		reviewChapter("One", first),
		reviewChapter("Two", []string{"A", "short", "chapter"}),
	}

	review := BuildReview(chapters, nil)
	if got := review.PageForChapter(0); got != 0 {
		t.Errorf("PageForChapter(0) = %d, want 0", got)
	}
	if got := review.PageForChapter(1); got != 2 {
		t.Errorf("PageForChapter(1) = %d, want 2", got)
	}
	if got := review.PageForChapter(99); got != 0 {
		t.Errorf("PageForChapter(99) = %d, want 0", got)
	}

	var missing *Review
	if got := missing.Percent(); got != 0 {
		t.Errorf("nil Percent() = %v, want 0", got)
	}
	if got := missing.PageForChapter(0); got != 0 {
		t.Errorf("nil PageForChapter() = %d, want 0", got)
	}
}

func TestBuildReviewWithoutChapters(t *testing.T) {
	if review := BuildReview(nil, nil); review != nil {
		t.Errorf("BuildReview(nil) = %+v, want nil", review)
	}
}
