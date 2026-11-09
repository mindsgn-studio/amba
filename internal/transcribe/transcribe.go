// Package transcribe turns an audiobook into timed words with sherpa-onnx's
// Whisper recogniser, and caches the result so it is paid for once.
package transcribe

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	sherpa "github.com/k2-fsa/sherpa-onnx-go/v2/sherpa_onnx"

	"github.com/mindsgn-studio/amba/internal/media"
	"github.com/mindsgn-studio/amba/internal/store"
)

const (
	sampleRate        = 16000
	chunkSeconds      = 28
	modelThreads      = 4
	transcriptVersion = 1
)

// Config names the model and the converter.
type Config struct {
	ModelDir  string
	ModelsDir string
	ModelURL  string
	Converter Converter
}

// Service transcribes books and owns the speech model.
type Service struct {
	modelDir  string
	modelsDir string
	modelURL  string
	converter Converter
	store     *store.Store

	modelMu sync.Mutex
}

// New creates a transcription service.
func New(config Config, cache *store.Store) *Service {
	converter := config.Converter
	if converter == nil {
		converter = FFmpegConverter{}
	}
	return &Service{
		modelDir:  config.ModelDir,
		modelsDir: config.ModelsDir,
		modelURL:  config.ModelURL,
		converter: converter,
		store:     cache,
	}
}

// Transcript returns a book's timed words, using the cache when the audio
// files have not changed and transcribing them otherwise.
func (s *Service) Transcript(
	ctx context.Context,
	bookID string,
	files []string,
	progress func(Progress),
) (*Transcript, error) {
	if cached, ok := s.loadCached(bookID, files); ok {
		report(progress, "Audiobook is ready.", 1)
		return cached, nil
	}

	if err := s.EnsureModel(ctx, progress); err != nil {
		return nil, err
	}

	entries := make([]File, len(files))
	var start time.Duration
	for i, filename := range files {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		info, err := os.Stat(filename)
		if err != nil {
			return nil, fmt.Errorf("read audio file %s: %w", filename, err)
		}
		duration, err := media.Duration(ctx, filename)
		if err != nil {
			return nil, err
		}
		entries[i] = File{
			Path:     filename,
			Size:     info.Size(),
			ModTime:  info.ModTime().UnixNano(),
			Duration: duration,
			Start:    start,
		}
		start += duration
	}

	report(progress, "Loading the speech recogniser…", 0)
	recognizer, err := s.createRecognizer()
	if err != nil {
		return nil, err
	}
	defer sherpa.DeleteOfflineRecognizer(recognizer)

	var words []Word
	for fileIndex, filename := range files {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		base := float64(fileIndex) / float64(len(files))
		share := 1 / float64(len(files))
		report(progress, fmt.Sprintf(
			"Transcribing chapter %d of %d…",
			fileIndex+1,
			len(files),
		), base)

		temporary, err := os.MkdirTemp("", "amba-audio-*")
		if err != nil {
			return nil, err
		}
		chunks, err := s.converter.Chunks(ctx, filename, temporary, chunkSeconds)
		if err != nil {
			os.RemoveAll(temporary)
			return nil, err
		}

		for chunkIndex, chunk := range chunks {
			if err := ctx.Err(); err != nil {
				os.RemoveAll(temporary)
				return nil, err
			}

			offset := entries[fileIndex].Start + time.Duration(chunkIndex*chunkSeconds)*time.Second
			chunkWords, err := recogniseChunk(recognizer, chunk, offset)
			if err != nil {
				os.RemoveAll(temporary)
				return nil, err
			}
			words = append(words, chunkWords...)

			fraction := base + share*(0.1+0.9*float64(chunkIndex+1)/float64(len(chunks)))
			report(progress, fmt.Sprintf(
				"Transcribing chapter %d of %d: %d%%",
				fileIndex+1,
				len(files),
				int(fraction*100),
			), fraction)
		}
		os.RemoveAll(temporary)
	}

	if len(words) == 0 {
		return nil, errors.New("the speech recogniser returned no words")
	}

	sort.SliceStable(words, func(i, j int) bool {
		return words[i].Start < words[j].Start
	})

	transcript := &Transcript{
		Version:  transcriptVersion,
		BookID:   bookID,
		Files:    entries,
		Words:    words,
		Duration: start,
	}
	if err := s.saveCache(transcript); err != nil {
		// A cache failure should not throw away a long transcription.
		report(progress, "Audiobook is ready, but its transcript could not be cached.", 1)
		return transcript, nil
	}

	report(progress, "Audiobook is ready.", 1)
	return transcript, nil
}

func (s *Service) createRecognizer() (*sherpa.OfflineRecognizer, error) {
	encoder := filepath.Join(s.modelDir, "tiny.en-encoder.int8.onnx")
	decoder := filepath.Join(s.modelDir, "tiny.en-decoder.int8.onnx")
	tokens := filepath.Join(s.modelDir, "tiny.en-tokens.txt")

	for _, filename := range []string{encoder, decoder, tokens} {
		if _, err := os.Stat(filename); err != nil {
			return nil, fmt.Errorf("speech model file is missing: %s", filename)
		}
	}

	config := sherpa.OfflineRecognizerConfig{
		FeatConfig: sherpa.FeatureConfig{
			SampleRate: sampleRate,
			FeatureDim: 80,
		},
		ModelConfig: sherpa.OfflineModelConfig{
			Whisper: sherpa.OfflineWhisperModelConfig{
				Encoder:                 encoder,
				Decoder:                 decoder,
				Language:                "en",
				Task:                    "transcribe",
				EnableTokenTimestamps:   1,
				EnableSegmentTimestamps: 1,
			},
			Tokens:     tokens,
			NumThreads: modelThreads,
			Provider:   "cpu",
		},
		DecodingMethod: "greedy_search",
	}

	recognizer := sherpa.NewOfflineRecognizer(&config)
	if recognizer == nil {
		return nil, errors.New("could not initialise the speech recogniser")
	}
	return recognizer, nil
}

func recogniseChunk(
	recognizer *sherpa.OfflineRecognizer,
	chunk string,
	offset time.Duration,
) ([]Word, error) {
	wave := sherpa.ReadWave(chunk)
	if wave == nil {
		return nil, fmt.Errorf("could not read audio chunk: %s", chunk)
	}
	if len(wave.Samples) == 0 {
		return nil, nil
	}

	stream := sherpa.NewOfflineStream(recognizer)
	if stream == nil {
		return nil, errors.New("could not create a speech stream")
	}
	defer sherpa.DeleteOfflineStream(stream)

	stream.AcceptWaveform(wave.SampleRate, wave.Samples)
	recognizer.Decode(stream)
	result := stream.GetResult()
	if result == nil {
		return nil, nil
	}

	// Models exported with cross-attention weights give exact token times.
	// The standard Whisper tiny.en export does not, so its text is placed in
	// time using the chunk's speech instead; the words are still in the right
	// order, which is what alignment with an EPUB needs.
	if len(result.Tokens) > 0 && len(result.Timestamps) > 0 {
		return tokensToWords(result.Tokens, result.Timestamps, result.Durations, offset), nil
	}
	return estimateWords(
		result.Text,
		offset,
		waveDuration(wave),
		speechSpans(wave.Samples, wave.SampleRate),
	), nil
}

func waveDuration(wave *sherpa.Wave) time.Duration {
	if wave.SampleRate <= 0 {
		return 0
	}
	return time.Duration(float64(len(wave.Samples)) / float64(wave.SampleRate) * float64(time.Second))
}

// tokensToWords groups Whisper's sub-word tokens into words and turns their
// chunk-relative timestamps into book-relative ones.
func tokensToWords(
	tokens []string,
	timestamps []float32,
	durations []float32,
	offset time.Duration,
) []Word {
	var words []Word
	var text strings.Builder
	var start, end time.Duration

	flush := func() {
		value := strings.TrimSpace(text.String())
		text.Reset()
		if value == "" {
			return
		}
		words = append(words, Word{Text: value, Start: start, End: end})
	}

	limit := len(tokens)
	if len(timestamps) < limit {
		limit = len(timestamps)
	}

	for i := 0; i < limit; i++ {
		token := tokens[i]
		if strings.HasPrefix(token, "<|") {
			continue
		}

		piece := strings.ReplaceAll(token, "\u2581", " ")
		piece = strings.ReplaceAll(piece, "\u0120", " ")
		startsNewWord := strings.HasPrefix(piece, " ")
		piece = strings.TrimSpace(piece)
		if piece == "" {
			continue
		}
		if startsNewWord && text.Len() > 0 {
			flush()
		}

		tokenStart := offset + time.Duration(float64(timestamps[i])*float64(time.Second))
		tokenDuration := 80 * time.Millisecond
		if i < len(durations) && durations[i] > 0 {
			tokenDuration = time.Duration(float64(durations[i]) * float64(time.Second))
		} else if i+1 < limit && timestamps[i+1] > timestamps[i] {
			tokenDuration = time.Duration(float64(timestamps[i+1]-timestamps[i]) * float64(time.Second))
		}

		if text.Len() == 0 {
			start = tokenStart
		}
		text.WriteString(piece)
		end = tokenStart + tokenDuration
	}
	flush()
	return words
}

func (s *Service) cachePath(bookID string) string {
	return s.store.File("books", bookID, "transcript.json")
}

func (s *Service) loadCached(bookID string, files []string) (*Transcript, bool) {
	var transcript Transcript
	if err := s.store.ReadJSON(s.cachePath(bookID), &transcript); err != nil {
		return nil, false
	}
	if transcript.Version != transcriptVersion || len(transcript.Words) == 0 {
		return nil, false
	}
	if len(transcript.Files) != len(files) {
		return nil, false
	}
	for i, filename := range files {
		info, err := os.Stat(filename)
		if err != nil {
			return nil, false
		}
		entry := transcript.Files[i]
		if entry.Path != filename || entry.Size != info.Size() || entry.ModTime != info.ModTime().UnixNano() {
			return nil, false
		}
	}
	return &transcript, true
}

func (s *Service) saveCache(transcript *Transcript) error {
	return s.store.WriteJSON(s.cachePath(transcript.BookID), transcript)
}
