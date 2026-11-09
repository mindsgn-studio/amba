package transcribe

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
)

// Converter turns an audio file into short, mono, 16 kHz WAV chunks that the
// speech recogniser can process.
type Converter interface {
	Chunks(ctx context.Context, input, outputDir string, seconds int) ([]string, error)
}

// FFmpegConverter converts with the ffmpeg command.
type FFmpegConverter struct{}

// Chunks converts input into numbered WAV chunks of at most seconds each.
func (FFmpegConverter) Chunks(ctx context.Context, input, outputDir string, seconds int) ([]string, error) {
	pattern := filepath.Join(outputDir, "chunk-%05d.wav")
	arguments := []string{
		"-hide_banner",
		"-loglevel", "error",
		"-nostdin",
		"-y",
		"-i", input,
		"-vn",
		"-ac", "1",
		"-ar", fmt.Sprint(sampleRate),
		"-c:a", "pcm_s16le",
		"-f", "segment",
		"-segment_time", fmt.Sprint(seconds),
		"-reset_timestamps", "1",
		"-segment_format", "wav",
		pattern,
	}

	var stderr bytes.Buffer
	command := exec.CommandContext(ctx, "ffmpeg", arguments...)
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		if errors.Is(ctx.Err(), context.Canceled) {
			return nil, ctx.Err()
		}
		return nil, fmt.Errorf("ffmpeg conversion failed: %w: %s", err, stderr.String())
	}

	chunks, err := filepath.Glob(filepath.Join(outputDir, "chunk-*.wav"))
	if err != nil {
		return nil, err
	}
	sort.Strings(chunks)
	if len(chunks) == 0 {
		return nil, errors.New("ffmpeg produced no audio chunks")
	}
	return chunks, nil
}
