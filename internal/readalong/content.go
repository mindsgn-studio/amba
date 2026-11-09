// Package readalong turns a book and its audiobook transcript into the timed
// text that the reader shows. It is the bridge between the printed page and
// the narration: every word on screen knows where it is heard.
package readalong

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/mindsgn-studio/amba/internal/align"
	"github.com/mindsgn-studio/amba/internal/epub"
	"github.com/mindsgn-studio/amba/internal/textutil"
	"github.com/mindsgn-studio/amba/internal/transcribe"
)

// MinSyncConfidence is kept for callers that predate chapter-level alignment.
// New code should use align.Config's thresholds instead.
const MinSyncConfidence = 0.25

// Content is a book laid out for reading, with the narration attached.
type Content struct {
	// Chapters are the pages of the book, in reading order. A rejected EPUB
	// chapter is replaced by a transcript chapter so reading can continue.
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
	// Synced reports that at least one chapter comes from the book and
	// matched its narration well. Without it the reader falls back to the
	// transcript.
	Synced bool
	// Confidence is the overall alignment quality score, from 0 to 1. It is
	// not a statistically calibrated probability.
	Confidence float64
	// EbookCoverage, AudioCoverage and Quality are the components of
	// Confidence.
	EbookCoverage float64
	AudioCoverage float64
	Quality       float64
	// Accepted reports whether the whole book passed the acceptance rules.
	Accepted bool
	// Diagnostics explains why the whole-book alignment was or was not
	// accepted.
	Diagnostics []string
	// Alignment is the persisted, cacheable alignment result.
	Alignment *Alignment
}

// Chapter is one page of the book.
type Chapter struct {
	Title      string
	Paragraphs []Paragraph
	// Index is the chapter's place in the book.
	Index int
	// Source is "epub" when the chapter shows the book's text, or
	// "transcript" when it is a fallback for an unsynchronised chapter.
	Source string
	// Synced reports that this chapter's words are mapped to narration
	// timestamps from the ebook alignment.
	Synced bool
	// Accepted and Rejected report the chapter-level alignment decision.
	Accepted bool
	Rejected bool
	// Confidence, EbookCoverage, AudioCoverage and Quality describe this
	// chapter's alignment.
	Confidence    float64
	EbookCoverage float64
	AudioCoverage float64
	Quality       float64
	// Diagnostics explains a rejection.
	Diagnostics []string
}

// Paragraph is a run of words that the reader keeps together.
type Paragraph struct {
	Tokens []Token
}

// Token is one word as printed, with punctuation, and the narration word
// that reads it. Audio is -1 when no reliable match exists; the reader never
// invents a timestamp for such a word.
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
// content. Alignment is evaluated chapter by chapter; rejected chapters fall
// back to the narration's transcript when other chapters are usable.
func FromEPUB(chapters []epub.Chapter, transcript *transcribe.Transcript) (*Content, error) {
	return FromEPUBConfig(chapters, transcript, align.DefaultConfig())
}

// FromEPUBConfig is FromEPUB with configurable acceptance thresholds.
func FromEPUBConfig(chapters []epub.Chapter, transcript *transcribe.Transcript, config align.Config) (*Content, error) {
	alignment, err := AlignBook(context.Background(), chapters, transcript, config, nil)
	if err != nil {
		return nil, err
	}
	return BuildContent(chapters, transcript, alignment)
}

// FromEPUBWithAlignment rebuilds content from a cached alignment without
// running the alignment engine again.
func FromEPUBWithAlignment(chapters []epub.Chapter, transcript *transcribe.Transcript, alignment *Alignment) (*Content, error) {
	return BuildContent(chapters, transcript, alignment)
}

// BuildContent turns an alignment into displayable, timed content.
func BuildContent(chapters []epub.Chapter, transcript *transcribe.Transcript, alignment *Alignment) (*Content, error) {
	if transcript == nil || len(transcript.Words) == 0 {
		return nil, errors.New("the audiobook has no transcript")
	}
	if len(chapters) == 0 {
		return nil, errors.New("the book has no readable chapters")
	}
	if alignment == nil {
		return nil, errors.New("the book has no alignment")
	}

	ebook := extractEbook(chapters)
	if len(ebook.chapters) == 0 {
		return nil, errors.New("the book has no readable words")
	}

	content := &Content{
		Audio:         transcript.Words,
		Duration:      durationOf(transcript),
		Confidence:    alignment.Confidence,
		EbookCoverage: alignment.EbookCoverage,
		AudioCoverage: alignment.AudioCoverage,
		Quality:       alignment.Quality,
		Accepted:      alignment.Accepted,
		Diagnostics:   alignment.Diagnostics,
		Alignment:     alignment,
	}

	matches := make(map[int]align.WordMatch, len(alignment.Matches))
	for _, match := range alignment.Matches {
		matches[match.EbookWordIndex] = match
	}

	anyAccepted := false
	for _, chapter := range alignment.Chapters {
		if chapter.Accepted {
			anyAccepted = true
			break
		}
	}
	content.Synced = anyAccepted

	targetLen := len(transcript.Words)
	fallbacks := fallbackAudioRanges(ebook.originals, alignment, targetLen)

	for index, chapter := range ebook.chapters {
		result := chapterResult(alignment, index)
		chapter.Index = index
		chapter.Source = "epub"
		applyChapterResult(&chapter, result)

		if result.Accepted {
			for sourceIndex := ebook.ranges[index].Start; sourceIndex < ebook.ranges[index].End; sourceIndex++ {
				match, ok := matches[sourceIndex]
				if !ok {
					continue
				}
				ref := ebook.sourceRefs[sourceIndex]
				chapter.Paragraphs[ref.Paragraph].Tokens[ref.Token].Audio = match.AudioWordIndex
			}
			content.Chapters = append(content.Chapters, chapter)
			continue
		}

		// Without any accepted chapter the whole book is unsynchronised and
		// the app will use the transcript. Keep the printed text intact so a
		// caller can still show it.
		if !anyAccepted {
			content.Chapters = append(content.Chapters, chapter)
			continue
		}

		fallback, ok := fallbacks[index]
		if !ok || fallback.end <= fallback.start {
			content.Chapters = append(content.Chapters, chapter)
			continue
		}
		content.Chapters = append(content.Chapters, transcriptChapter(
			chapter.Title,
			transcript,
			fallback.start,
			fallback.end,
			index,
			result,
		))
	}

	reindexContent(content)
	return content, nil
}

func chapterResult(alignment *Alignment, index int) align.ChapterResult {
	if index < 0 || index >= len(alignment.Chapters) {
		return align.ChapterResult{Index: index}
	}
	chapter := alignment.Chapters[index]
	return align.ChapterResult{
		Index:      chapter.Index,
		Range:      chapter.Range,
		AudioStart: chapter.AudioStart,
		AudioEnd:   chapter.AudioEnd,
		AlignmentResult: align.AlignmentResult{
			Accepted:      chapter.Accepted,
			Confidence:    chapter.Confidence,
			EbookCoverage: chapter.EbookCoverage,
			AudioCoverage: chapter.AudioCoverage,
			Quality:       chapter.Quality,
			Sequential:    chapter.Sequential,
			Diagnostics:   chapter.Diagnostics,
		},
	}
}

func applyChapterResult(chapter *Chapter, result align.ChapterResult) {
	chapter.Accepted = result.Accepted
	chapter.Rejected = !result.Accepted
	chapter.Synced = result.Accepted
	chapter.Confidence = result.Confidence
	chapter.EbookCoverage = result.EbookCoverage
	chapter.AudioCoverage = result.AudioCoverage
	chapter.Quality = result.Quality
	chapter.Diagnostics = result.Diagnostics
}

type audioRange struct {
	start int
	end   int
}

// fallbackAudioRanges divides the narration between accepted chapters among
// the rejected chapters in each gap. It uses accepted chapters as anchors and
// allocates the gap in proportion to each rejected chapter's word count.
func fallbackAudioRanges(chapters []epub.Chapter, alignment *Alignment, targetLen int) map[int]audioRange {
	ranges := make(map[int]audioRange)
	if len(chapters) == 0 || alignment == nil || targetLen <= 0 {
		return ranges
	}

	type boundary struct {
		index int
		start int
		end   int
	}
	var accepted []boundary
	for index, chapter := range alignment.Chapters {
		if !chapter.Accepted || index < 0 || index >= len(chapters) {
			continue
		}
		start, end := chapter.AudioStart, chapter.AudioEnd
		if start < 0 || end <= start {
			continue
		}
		if end > targetLen {
			end = targetLen
		}
		accepted = append(accepted, boundary{index: index, start: start, end: end})
	}
	if len(accepted) == 0 {
		return ranges
	}

	allocate := func(from, to int, gapStart, gapEnd int) {
		if to <= from {
			return
		}
		if gapStart < 0 {
			gapStart = 0
		}
		if gapEnd > targetLen {
			gapEnd = targetLen
		}
		if gapEnd < gapStart {
			gapEnd = gapStart
		}
		lengths := make([]int, to-from)
		total := 0
		for index := from; index < to; index++ {
			length := chapterWordCount(chapters[index])
			if length < 1 {
				length = 1
			}
			lengths[index-from] = length
			total += length
		}
		cursor := gapStart
		for offset, length := range lengths {
			next := cursor + (gapEnd-gapStart)*length/total
			if offset == len(lengths)-1 {
				next = gapEnd
			}
			ranges[from+offset] = audioRange{start: cursor, end: next}
			cursor = next
		}
	}

	allocate(0, accepted[0].index, 0, accepted[0].start)
	for i := 0; i+1 < len(accepted); i++ {
		allocate(accepted[i].index+1, accepted[i+1].index, accepted[i].end, accepted[i+1].start)
	}
	allocate(accepted[len(accepted)-1].index+1, len(chapters), accepted[len(accepted)-1].end, targetLen)
	return ranges
}

func chapterWordCount(chapter epub.Chapter) int {
	count := 0
	for _, paragraph := range chapter.Paragraphs {
		for _, word := range paragraph.Words {
			if textutil.IsWord(word) {
				count++
			}
		}
	}
	return count
}

func transcriptChapter(title string, transcript *transcribe.Transcript, start, end, index int, result align.ChapterResult) Chapter {
	if start < 0 {
		start = 0
	}
	if end > len(transcript.Words) {
		end = len(transcript.Words)
	}
	chapter := Chapter{
		Title:      title,
		Index:      index,
		Source:     "transcript",
		Synced:     false,
		Accepted:   false,
		Rejected:   true,
		Confidence: result.Confidence,
		Diagnostics: append([]string{
			"EPUB synchronisation is unavailable for this chapter; showing the audiobook transcript.",
		}, result.Diagnostics...),
	}
	if chapter.Title == "" {
		chapter.Title = fmt.Sprintf("Chapter %d", index+1)
	}

	var paragraph Paragraph
	var previousStart time.Duration
	flush := func() {
		if len(paragraph.Tokens) > 0 {
			chapter.Paragraphs = append(chapter.Paragraphs, paragraph)
			paragraph = Paragraph{}
		}
	}
	for wordIndex := start; wordIndex < end; wordIndex++ {
		word := transcript.Words[wordIndex]
		if len(paragraph.Tokens) >= 90 || (len(paragraph.Tokens) > 0 && word.Start-previousStart > 1200*time.Millisecond) {
			flush()
		}
		paragraph.Tokens = append(paragraph.Tokens, Token{
			Text:       word.Text,
			Normalized: textutil.Normalize(word.Text),
			Audio:      wordIndex,
		})
		previousStart = word.Start
	}
	flush()
	if len(chapter.Paragraphs) == 0 {
		chapter.Paragraphs = []Paragraph{{}}
	}
	return chapter
}

// ebookData is the printed book laid out for alignment.
type ebookData struct {
	chapters    []Chapter
	originals   []epub.Chapter
	flat        []FlatToken
	sourceWords []string
	sourceRefs  []FlatToken
	ranges      []align.Range
}

func extractEbook(chapters []epub.Chapter) ebookData {
	var data ebookData
	for _, chapter := range chapters {
		index := len(data.chapters)
		display := Chapter{Title: chapter.Title, Index: index, Source: "epub"}
		if display.Title == "" {
			display.Title = fmt.Sprintf("Chapter %d", index+1)
		}

		start := len(data.sourceWords)
		for _, paragraph := range chapter.Paragraphs {
			paragraphIndex := len(display.Paragraphs)
			displayParagraph := Paragraph{}

			for _, word := range paragraph.Words {
				normalized := textutil.Normalize(word)
				flat := len(data.flat)
				displayParagraph.Tokens = append(displayParagraph.Tokens, Token{
					Text:       word,
					Normalized: normalized,
					Audio:      -1,
				})
				ref := FlatToken{
					Flat:      flat,
					Chapter:   index,
					Paragraph: paragraphIndex,
					Token:     len(displayParagraph.Tokens) - 1,
				}
				data.flat = append(data.flat, ref)

				if normalized != "" && textutil.IsWord(word) {
					data.sourceWords = append(data.sourceWords, normalized)
					data.sourceRefs = append(data.sourceRefs, ref)
				}
			}

			if len(displayParagraph.Tokens) > 0 {
				display.Paragraphs = append(display.Paragraphs, displayParagraph)
			}
		}

		if len(display.Paragraphs) > 0 {
			data.chapters = append(data.chapters, display)
			data.originals = append(data.originals, chapter)
			data.ranges = append(data.ranges, align.Range{Start: start, End: len(data.sourceWords)})
		}
	}
	return data
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
		chapter := Chapter{
			Title:  fmt.Sprintf("Part %d", chapterIndex+1),
			Index:  chapterIndex,
			Source: "transcript",
		}
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

			paragraph.Tokens = append(paragraph.Tokens, Token{
				Text:       word.Text,
				Normalized: textutil.Normalize(word.Text),
				Audio:      index,
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
		content.Chapters = []Chapter{{Title: "Audiobook", Index: 0, Source: "transcript"}}
		chapter := &content.Chapters[0]
		for index, word := range transcript.Words {
			chapter.Paragraphs = append(chapter.Paragraphs, Paragraph{Tokens: []Token{{
				Text:       word.Text,
				Normalized: textutil.Normalize(word.Text),
				Audio:      index,
			}}})
		}
	}

	reindexContent(content)
	return content
}

// reindexContent builds Flat, ByAudio and ByAudioChapter from Chapters.
func reindexContent(content *Content) {
	content.Flat = content.Flat[:0]
	for chapterIndex, chapter := range content.Chapters {
		for paragraphIndex, paragraph := range chapter.Paragraphs {
			for tokenIndex := range paragraph.Tokens {
				content.Flat = append(content.Flat, FlatToken{
					Flat:      len(content.Flat),
					Chapter:   chapterIndex,
					Paragraph: paragraphIndex,
					Token:     tokenIndex,
				})
			}
		}
	}

	content.ByAudio = make([]int, len(content.Audio))
	content.ByAudioChapter = make([]int, len(content.Audio))
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
