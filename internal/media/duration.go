// Package media reads facts about audio files that playback needs before it
// starts, without decoding them.
package media

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Duration asks ffprobe for an audio file's length. ffprobe reads the
// container's headers instead of decoding the whole file, which matters for
// audiobooks of hundreds of megabytes.
func Duration(ctx context.Context, filename string) (time.Duration, error) {
	command := exec.CommandContext(
		ctx,
		"ffprobe",
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		filename,
	)
	output, err := command.Output()
	if err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			return 0, fmt.Errorf("ffprobe could not read %s: %s", filename, strings.TrimSpace(string(exitError.Stderr)))
		}
		return 0, fmt.Errorf("ffprobe is required to read audio durations: %w", err)
	}

	seconds, err := strconv.ParseFloat(strings.TrimSpace(string(output)), 64)
	if err != nil || seconds <= 0 {
		return 0, fmt.Errorf("ffprobe returned an invalid duration for %s", filename)
	}
	return time.Duration(seconds * float64(time.Second)), nil
}
