// Package readalong turns a book and its audiobook transcript into the timed
// text that the reader shows. It is the bridge between the printed page and
// the narration: every word on screen knows where it is heard.
package readalong

import (
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/mindsgn-studio/amba/internal/align"
	"github.com/mindsgn-studio/amba/internal/epub"
	"github.com/mindsgn-studio/amba/internal/textutil"
	"github.com/mindsgn-studio/amba/internal/transcribe"
)

// MinSyncConfidence is the share of a book's words that must match the
// transcript before the reader trusts the alignment. A lower share usually
// means the EPUB is a different edition, and the transcript reads better.
const MinSyncConfidence = 0.25

// Content is a book laid out for reading, with the narration attached.
type Content struct {
	// Chapters are the pages of the book, in reading order.
	Chapters []Chapter
	// Audio is the narration's timed words, in hearing order.
	Audio []transcribe.Word
	// Flat lists every displayed word in reading order, with its place in
	// Chapters.
	Flat []FlatToken
	// ByAudio maps a narration word to the displayed word it is reading,
	// or -1 before the first one matches. It is filled forward, so a
	// narration passage missing from the book keeps the last word lit.
	ByAudio []int
	// ByAudioChapter maps a narration word to the chapter it belongs to,
	// filled forward in the same way.
	ByAudioChapter []int
	// Duration is the length of the narration.
	Duration time.Duration
	// Synced reports that the chapters come from the book and matched its
	// narration well. Without it the chapters are the transcript itself.
	Synced bool
	// Confidence is the share of the book's words that matched the
	// narration, from 0 to 1.
	Confidence float64
}

// Chapter is one page of the book.
type Chapter struct {
	Title      string
	Paragraphs []Paragraph
}

// Paragraph is a run of words that the reader keeps together.
type Paragraph struct {
	Tokens []Token
}

// Token is one word as printed, with punctuation, and the narration word
// that reads it.
type Token struct {
	Text       string
	Normalized string
	Audio      int
}

// FlatToken locates a displayed word in Chapters.
type FlatToken struct {
	Flat      int
	Chapter   int
	Paragraph int
	Token     int
}

// FromEPUB matches a book's chapters to the narration and returns timed
// content. The content is still useful when confidence is low: Synced says
// whether the reader should trust it.
func FromEPUB(chapters []epub.Chapter, transcript *transcribe.Transcript) (*Content, error) {
	if transcript == nil || len(transcript.Words) == 0 {
		return nil, errors.New("the audiobook has no transcript")
	}
	if len(chapters) == 0 {
		return nil, errors.New("the book has no readable chapters")
	}

	content := &Content{
		Audio:    transcript.Words,
		Duration: durationOf(transcript),
	}

	// Build the printed words and the sequence that alignment compares.
	var sourceWords []string
	var sourceRefs []FlatToken

	for _, chapter := range chapters {
		chapterIndex := len(content.Chapters)
		display := Chapter{Title: chapter.Title}
		if display.Title == "" {
			display.Title = fmt.Sprintf("Chapter %d", chapterIndex+1)
		}

		for _, paragraph := range chapter.Paragraphs {
			paragraphIndex := len(display.Paragraphs)
			displayParagraph := Paragraph{}

			for _, word := range paragraph.Words {
				normalized := textutil.Normalize(word)
				token := Token{Text: word, Normalized: normalized, Audio: -1}
				flat := len(content.Flat)
				displayParagraph.Tokens = append(displayParagraph.Tokens, token)
				content.Flat = append(content.Flat, FlatToken{
					Flat:      flat,
					Chapter:   chapterIndex,
					Paragraph: paragraphIndex,
					Token:     len(displayParagraph.Tokens) - 1,
				})

				if normalized != "" && textutil.IsWord(word) {
					sourceWords = append(sourceWords, normalized)
					sourceRefs = append(sourceRefs, FlatToken{
						Flat:      flat,
						Chapter:   chapterIndex,
						Paragraph: paragraphIndex,
						Token:     len(displayParagraph.Tokens) - 1,
					})
				}
			}

			if len(displayParagraph.Tokens) > 0 {
				display.Paragraphs = append(display.Paragraphs, displayParagraph)
			}
		}

		if len(display.Paragraphs) > 0 {
			content.Chapters = append(content.Chapters, display)
		}
	}

	if len(content.Chapters) == 0 {
		return nil, errors.New("the book has no readable words")
	}

	// The narration's comparable words, and where each came from.
	targetWords := make([]string, 0, len(transcript.Words))
	targetAudio := make([]int, 0, len(transcript.Words))
	for index, word := range transcript.Words {
		normalized := textutil.Normalize(word.Text)
		if normalized == "" {
			continue
		}
		targetWords = append(targetWords, normalized)
		targetAudio = append(targetAudio, index)
	}

	result := align.Sequences(sourceWords, targetWords)

	targetToFlat := make([]int, len(targetWords))
	for i := range targetToFlat {
		targetToFlat[i] = -1
	}

	for sourceIndex, targetIndex := range result.Match {
		if targetIndex < 0 || targetIndex >= len(targetToFlat) {
			continue
		}
		ref := sourceRefs[sourceIndex]
		audioIndex := targetAudio[targetIndex]
		content.Chapters[ref.Chapter].Paragraphs[ref.Paragraph].Tokens[ref.Token].Audio = audioIndex
		if targetToFlat[targetIndex] < 0 {
			targetToFlat[targetIndex] = ref.Flat
		}
	}

	content.ByAudio = make([]int, len(transcript.Words))
	content.ByAudioChapter = make([]int, len(transcript.Words))
	for i := range content.ByAudio {
		content.ByAudio[i] = -1
		content.ByAudioChapter[i] = -1
	}
	for targetIndex, flat := range targetToFlat {
		if flat < 0 {
			continue
		}
		audioIndex := targetAudio[targetIndex]
		content.ByAudio[audioIndex] = flat
		content.ByAudioChapter[audioIndex] = content.Flat[flat].Chapter
	}
	fillForward(content.ByAudio)
	fillForward(content.ByAudioChapter)

	fillTokenAudio(content)
	content.Confidence = result.Confidence
	content.Synced = result.Confidence >= MinSyncConfidence
	return content, nil
}

// FromTranscript builds readable content from the narration alone. It is
// used when the book is missing, or when its text does not match the
// narration well enough to highlight it.
func FromTranscript(transcript *transcribe.Transcript) *Content {
	content := &Content{
		Audio:    transcript.Words,
		Duration: durationOf(transcript),
		Synced:   false,
	}
	if len(transcript.Words) == 0 {
		return content
	}

	files := transcript.Files
	if len(files) == 0 {
		files = []transcribe.File{{Duration: durationOf(transcript)}}
	}

	cursor := 0
	for fileIndex, file := range files {
		first := sort.Search(len(transcript.Words), func(i int) bool {
			return transcript.Words[i].Start >= file.Start
		})
		if first < cursor {
			first = cursor
		}

		last := len(transcript.Words)
		if file.Duration > 0 {
			end := file.Start + file.Duration
			last = sort.Search(len(transcript.Words), func(i int) bool {
				return transcript.Words[i].Start >= end
			})
		} else if fileIndex+1 < len(files) {
			last = first
		}
		if last < first {
			last = first
		}
		if first >= last {
			continue
		}

		chapterIndex := len(content.Chapters)
		chapter := Chapter{Title: fmt.Sprintf("Part %d", chapterIndex+1)}
		var paragraph Paragraph
		var previousStart time.Duration

		flush := func() {
			if len(paragraph.Tokens) > 0 {
				chapter.Paragraphs = append(chapter.Paragraphs, paragraph)
				paragraph = Paragraph{}
			}
		}

		for index := first; index < last; index++ {
			word := transcript.Words[index]
			if len(paragraph.Tokens) >= 90 || (len(paragraph.Tokens) > 0 && word.Start-previousStart > 1200*time.Millisecond) {
				flush()
			}

			paragraphIndex := len(chapter.Paragraphs)
			flat := len(content.Flat)
			paragraph.Tokens = append(paragraph.Tokens, Token{
				Text:       word.Text,
				Normalized: textutil.Normalize(word.Text),
				Audio:      index,
			})
			content.Flat = append(content.Flat, FlatToken{
				Flat:      flat,
				Chapter:   chapterIndex,
				Paragraph: paragraphIndex,
				Token:     len(paragraph.Tokens) - 1,
			})
			previousStart = word.Start
		}
		flush()

		if len(chapter.Paragraphs) > 0 {
			content.Chapters = append(content.Chapters, chapter)
		}
		cursor = last
	}

	if len(content.Chapters) == 0 {
		content.Chapters = []Chapter{{Title: "Audiobook"}}
		chapter := &content.Chapters[0]
		for index, word := range transcript.Words {
			chapter.Paragraphs = append(chapter.Paragraphs, Paragraph{Tokens: []Token{{
				Text:       word.Text,
				Normalized: textutil.Normalize(word.Text),
				Audio:      index,
			}}})
			content.Flat = append(content.Flat, FlatToken{
				Flat:      len(content.Flat),
				Chapter:   0,
				Paragraph: len(chapter.Paragraphs) - 1,
			})
		}
	}

	content.ByAudio = make([]int, len(transcript.Words))
	content.ByAudioChapter = make([]int, len(transcript.Words))
	for i := range content.ByAudio {
		content.ByAudio[i] = -1
		content.ByAudioChapter[i] = -1
	}
	for _, ref := range content.Flat {
		token := content.Chapters[ref.Chapter].Paragraphs[ref.Paragraph].Tokens[ref.Token]
		if token.Audio < 0 || token.Audio >= len(content.Audio) {
			continue
		}
		if content.ByAudio[token.Audio] < 0 {
			content.ByAudio[token.Audio] = ref.Flat
		}
		if content.ByAudioChapter[token.Audio] < 0 {
			content.ByAudioChapter[token.Audio] = ref.Chapter
		}
	}
	fillForward(content.ByAudio)
	fillForward(content.ByAudioChapter)

	return content
}

// ActiveAudio returns the narration word being heard at position, or -1
// before the narration starts.
func (c *Content) ActiveAudio(position time.Duration) int {
	if len(c.Audio) == 0 {
		return -1
	}
	index := sort.Search(len(c.Audio), func(i int) bool {
		return c.Audio[i].Start > position
	}) - 1
	if index < 0 {
		return -1
	}
	return index
}

// ActiveToken returns the displayed word being read at position, or -1
// when no word is highlighted.
func (c *Content) ActiveToken(position time.Duration) int {
	audio := c.ActiveAudio(position)
	if audio < 0 || audio >= len(c.ByAudio) {
		return -1
	}
	return c.ByAudio[audio]
}

// ChapterAt returns the chapter being read at position.
func (c *Content) ChapterAt(position time.Duration) int {
	if len(c.Chapters) == 0 {
		return 0
	}
	audio := c.ActiveAudio(position)
	if audio < 0 || audio >= len(c.ByAudioChapter) {
		return 0
	}
	chapter := c.ByAudioChapter[audio]
	if chapter < 0 || chapter >= len(c.Chapters) {
		return 0
	}
	return chapter
}

// ChapterStart returns where the narration begins reading chapter.
func (c *Content) ChapterStart(chapter int) time.Duration {
	if chapter < 0 || chapter >= len(c.Chapters) {
		return 0
	}
	for index, value := range c.ByAudioChapter {
		if value == chapter {
			return c.Audio[index].Start
		}
	}
	return 0
}

// TokenAt returns the displayed word at a flat index.
func (c *Content) TokenAt(flat int) (Token, bool) {
	if flat < 0 || flat >= len(c.Flat) {
		return Token{}, false
	}
	ref := c.Flat[flat]
	if ref.Chapter < 0 || ref.Chapter >= len(c.Chapters) {
		return Token{}, false
	}
	chapter := c.Chapters[ref.Chapter]
	if ref.Paragraph < 0 || ref.Paragraph >= len(chapter.Paragraphs) {
		return Token{}, false
	}
	paragraph := chapter.Paragraphs[ref.Paragraph]
	if ref.Token < 0 || ref.Token >= len(paragraph.Tokens) {
		return Token{}, false
	}
	return paragraph.Tokens[ref.Token], true
}

// fillTokenAudio gives punctuation and unmatched words the narration index
// of a neighbour, so every displayed word has a place in the timeline.
func fillTokenAudio(content *Content) {
	previous := -1
	for _, ref := range content.Flat {
		token := &content.Chapters[ref.Chapter].Paragraphs[ref.Paragraph].Tokens[ref.Token]
		if token.Audio >= 0 {
			previous = token.Audio
			continue
		}
		if previous >= 0 {
			token.Audio = previous
		}
	}

	next := -1
	for index := len(content.Flat) - 1; index >= 0; index-- {
		ref := content.Flat[index]
		token := &content.Chapters[ref.Chapter].Paragraphs[ref.Paragraph].Tokens[ref.Token]
		if token.Audio >= 0 {
			next = token.Audio
			continue
		}
		if next >= 0 {
			token.Audio = next
		}
	}
}

// fillForward replaces each -1 with the value before it, and leading -1s
// with the first value that follows.
func fillForward(values []int) {
	previous := -1
	first := -1
	for index, value := range values {
		if value >= 0 {
			previous = value
			if first < 0 {
				first = value
			}
			continue
		}
		if previous >= 0 {
			values[index] = previous
		}
	}
	for index := 0; index < len(values) && values[index] < 0; index++ {
		values[index] = first
	}
	for index := range values {
		if values[index] < 0 {
			values[index] = 0
		}
	}
}

func durationOf(transcript *transcribe.Transcript) time.Duration {
	if transcript.Duration > 0 {
		return transcript.Duration
	}
	var end time.Duration
	for _, word := range transcript.Words {
		if word.End > end {
			end = word.End
		}
	}
	return end
}
