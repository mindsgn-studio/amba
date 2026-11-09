package readalong

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/mindsgn-studio/amba/internal/align"
	"github.com/mindsgn-studio/amba/internal/epub"
	"github.com/mindsgn-studio/amba/internal/textutil"
	"github.com/mindsgn-studio/amba/internal/transcribe"
)

// AlignmentVersion identifies the meaning of cached alignment results.
const AlignmentVersion = align.AlgorithmVersion

// Alignment is the persisted result of matching an EPUB to an audiobook
// transcript. Its Key covers the transcript, the ebook text and the alignment
// configuration, so any change invalidates the cache.
type Alignment struct {
	Version        int    `json:"version"`
	Key            string `json:"key"`
	TranscriptHash string `json:"transcriptHash"`
	EbookHash      string `json:"ebookHash"`
	ConfigHash     string `json:"configHash"`

	align.AlignmentResult

	Chapters []ChapterAlignment `json:"chapters,omitempty"`
}

// ChapterAlignment is one chapter's part of a persisted alignment.
type ChapterAlignment struct {
	Index      int         `json:"index"`
	Title      string      `json:"title"`
	Range      align.Range `json:"range"`
	AudioStart int         `json:"audioStart"`
	AudioEnd   int         `json:"audioEnd"`

	align.AlignmentResult
}

// AlignBook aligns an EPUB to an audiobook transcript and returns a result
// that can be cached. progress receives fractions from 0 to 1.
func AlignBook(
	ctx context.Context,
	chapters []epub.Chapter,
	transcript *transcribe.Transcript,
	config align.Config,
	progress func(float64),
) (*Alignment, error) {
	if transcript == nil || len(transcript.Words) == 0 {
		return nil, errors.New("the audiobook has no transcript")
	}
	if len(chapters) == 0 {
		return nil, errors.New("the book has no readable chapters")
	}

	ebook := extractEbook(chapters)
	if len(ebook.chapters) == 0 || len(ebook.sourceWords) == 0 {
		return nil, errors.New("the book has no readable words")
	}

	targetWords := make([]string, 0, len(transcript.Words))
	targetAudio := make([]int, 0, len(transcript.Words))
	starts := make([]float64, 0, len(transcript.Words))
	ends := make([]float64, 0, len(transcript.Words))
	for index, word := range transcript.Words {
		normalized := textutil.Normalize(word.Text)
		if normalized == "" {
			continue
		}
		targetWords = append(targetWords, normalized)
		targetAudio = append(targetAudio, index)
		starts = append(starts, word.Start.Seconds())
		ends = append(ends, word.End.Seconds())
	}
	if len(targetWords) == 0 {
		return nil, errors.New("the audiobook transcript has no words")
	}

	book, err := align.AlignChaptersContext(
		ctx,
		ebook.sourceWords,
		targetWords,
		ebook.ranges,
		starts,
		ends,
		config,
		progress,
	)
	if err != nil {
		return nil, err
	}

	// Alignment indices are into targetWords; convert them back to indices
	// into the transcript's full word list.
	book.Matches = convertMatches(book.Matches, targetAudio)
	for index := range book.Chapters {
		chapter := &book.Chapters[index]
		chapter.Matches = convertMatches(chapter.Matches, targetAudio)
		chapter.AudioStart, chapter.AudioEnd = convertRange(chapter.AudioStart, chapter.AudioEnd, targetAudio)
	}

	alignment := &Alignment{
		Version:         AlignmentVersion,
		AlignmentResult: book.AlignmentResult,
	}
	for index, chapter := range book.Chapters {
		title := ""
		if index < len(ebook.chapters) {
			title = ebook.chapters[index].Title
		}
		alignment.Chapters = append(alignment.Chapters, ChapterAlignment{
			Index:      index,
			Title:      title,
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
				Matches:       chapter.Matches,
				Diagnostics:   chapter.Diagnostics,
			},
		})
	}

	alignment.TranscriptHash = HashTranscript(transcript)
	alignment.EbookHash = HashEbook(chapters)
	alignment.ConfigHash = HashConfig(config)
	alignment.Key = AlignmentKeyFromHashes(alignment.TranscriptHash, alignment.EbookHash, alignment.ConfigHash)
	return alignment, nil
}

// AlignmentKey returns the cache key for an ebook, transcript and
// configuration.
func AlignmentKey(chapters []epub.Chapter, transcript *transcribe.Transcript, config align.Config) string {
	return AlignmentKeyFromHashes(HashTranscript(transcript), HashEbook(chapters), HashConfig(config))
}

// AlignmentKeyFromHashes combines the three component hashes.
func AlignmentKeyFromHashes(transcriptHash, ebookHash, configHash string) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%d\x00%s\x00%s\x00%s", AlignmentVersion, transcriptHash, ebookHash, configHash)))
	return hex.EncodeToString(sum[:])
}

// HashTranscript hashes the timed words that alignment depends on.
func HashTranscript(transcript *transcribe.Transcript) string {
	hash := sha256.New()
	if transcript == nil {
		return hex.EncodeToString(hash.Sum(nil))
	}
	for _, word := range transcript.Words {
		writeHashString(hash, word.Text)
		writeHashInt(hash, int64(word.Start))
		writeHashInt(hash, int64(word.End))
	}
	return hex.EncodeToString(hash.Sum(nil))
}

// HashEbook hashes the extracted ebook text and its chapter/paragraph
// boundaries. A changed EPUB therefore invalidates cached alignments.
func HashEbook(chapters []epub.Chapter) string {
	hash := sha256.New()
	for _, chapter := range chapters {
		writeHashString(hash, chapter.Title)
		for _, paragraph := range chapter.Paragraphs {
			writeHashString(hash, paragraph.Text)
			for _, word := range paragraph.Words {
				writeHashString(hash, word)
			}
		}
	}
	return hex.EncodeToString(hash.Sum(nil))
}

// HashConfig hashes the canonical alignment configuration and version.
func HashConfig(config align.Config) string {
	canonical := config.Canonical()
	data, err := json.Marshal(struct {
		Version int
		Config  align.Config
	}{Version: AlignmentVersion, Config: canonical})
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func writeHashString(hash interface{ Write([]byte) (int, error) }, value string) {
	var size [8]byte
	binary.BigEndian.PutUint64(size[:], uint64(len(value)))
	hash.Write(size[:])
	hash.Write([]byte(value))
}

func writeHashInt(hash interface{ Write([]byte) (int, error) }, value int64) {
	var buffer [8]byte
	binary.BigEndian.PutUint64(buffer[:], uint64(value))
	hash.Write(buffer[:])
}

func convertMatches(matches []align.WordMatch, targetAudio []int) []align.WordMatch {
	if len(matches) == 0 {
		return nil
	}
	converted := make([]align.WordMatch, 0, len(matches))
	for _, match := range matches {
		if match.AudioWordIndex < 0 || match.AudioWordIndex >= len(targetAudio) {
			continue
		}
		match.AudioWordIndex = targetAudio[match.AudioWordIndex]
		converted = append(converted, match)
	}
	return converted
}

func convertRange(start, end int, targetAudio []int) (int, int) {
	if start < 0 || start >= len(targetAudio) {
		return -1, -1
	}
	if end <= start {
		return targetAudio[start], targetAudio[start] + 1
	}
	if end > len(targetAudio) {
		end = len(targetAudio)
	}
	return targetAudio[start], targetAudio[end-1] + 1
}
