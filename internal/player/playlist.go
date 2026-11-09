// Package player plays an audiobook made of several files as one timeline.
package player

import (
	"errors"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/egoist/mygo/audio"
)

// Track is one audio file and the length of its part of the book.
type Track struct {
	Path     string
	Duration time.Duration
}

// Playlist plays tracks one after another. Its position and duration are those
// of the whole book, not of the current file, so the reader can treat the
// audiobook as a single timeline.
type Playlist struct {
	mu sync.Mutex

	tracks []Track
	starts []time.Duration
	total  time.Duration

	index   int
	player  *audio.Player
	playing bool
	speed   float64
	volume  float64
	closed  bool
	err     error
}

// ErrClosed is returned when a closed playlist is played.
var ErrClosed = errors.New("playlist is closed")

// New creates a playlist. Tracks with a zero duration still play, but do not
// advance the timeline correctly, so callers should fill every duration.
func New(tracks []Track) *Playlist {
	starts := make([]time.Duration, len(tracks))
	var total time.Duration
	for i, track := range tracks {
		starts[i] = total
		if track.Duration > 0 {
			total += track.Duration
		}
	}
	return &Playlist{
		tracks: tracks,
		starts: starts,
		total:  total,
		speed:  1,
		volume: 1,
	}
}

// Duration is the length of the whole book.
func (p *Playlist) Duration() time.Duration {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.total
}

// Position is the place in the whole book that is being heard.
func (p *Playlist) Position() time.Duration {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.tracks) == 0 {
		return 0
	}
	if p.player == nil {
		return p.starts[p.index]
	}
	position := p.starts[p.index] + p.player.Position()
	if position < 0 {
		return 0
	}
	if position > p.total {
		return p.total
	}
	return position
}

// Playing reports whether playback is wanted.
func (p *Playlist) Playing() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.playing
}

// Err is the last playback error.
func (p *Playlist) Err() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.err
}

// Play starts or resumes playback.
func (p *Playlist) Play() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return ErrClosed
	}
	if len(p.tracks) == 0 {
		return errors.New("the audiobook has no tracks")
	}
	if p.playing {
		return nil
	}
	if p.index >= len(p.tracks) {
		p.index = 0
	}
	if p.player == nil {
		if err := p.openLocked(p.index); err != nil {
			p.err = err
			return err
		}
	}
	p.player.Play()
	p.playing = true
	return nil
}

// Pause pauses playback.
func (p *Playlist) Pause() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.player != nil {
		p.player.Pause()
	}
	p.playing = false
}

// Toggle plays or pauses.
func (p *Playlist) Toggle() error {
	if p.Playing() {
		p.Pause()
		return nil
	}
	return p.Play()
}

// Seek moves to a place in the whole book.
func (p *Playlist) Seek(position time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed || len(p.tracks) == 0 {
		return
	}
	if position < 0 {
		position = 0
	}
	if position > p.total {
		position = p.total
	}

	index := p.indexAtLocked(position)
	local := position - p.starts[index]
	if p.player != nil && p.index == index {
		p.player.Seek(local)
		return
	}
	if p.player != nil {
		p.player.Close()
		p.player = nil
	}
	if err := p.openLocked(index); err != nil {
		p.err = err
		p.playing = false
		return
	}
	p.player.Seek(local)
	if p.playing {
		p.player.Play()
	}
}

// SetSpeed changes the playback speed, preserving the narrator's pitch.
func (p *Playlist) SetSpeed(speed float64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if speed <= 0 {
		return
	}
	p.speed = speed
	if p.player != nil {
		p.player.SetPreservesPitch(true).SetSpeed(speed)
	}
}

// Speed is the current playback speed.
func (p *Playlist) Speed() float64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.speed
}

// SetVolume sets the playback volume between 0 and 1.
func (p *Playlist) SetVolume(volume float64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if volume < 0 {
		volume = 0
	}
	if volume > 1 {
		volume = 1
	}
	p.volume = volume
	if p.player != nil {
		p.player.SetVolume(volume)
	}
}

// Close stops playback and releases the current file.
func (p *Playlist) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return
	}
	p.closed = true
	p.playing = false
	if p.player != nil {
		p.player.Stop()
		p.player.Close()
		p.player = nil
	}
}

// openLocked opens the track at index. The caller holds p.mu.
func (p *Playlist) openLocked(index int) error {
	if index < 0 || index >= len(p.tracks) {
		return errors.New("track index out of range")
	}

	file, err := os.Open(p.tracks[index].Path)
	if err != nil {
		return err
	}

	stream, err := audio.OpenStream(file)
	if err != nil {
		file.Close()
		return err
	}

	p.player = stream
	p.index = index
	p.err = nil

	stream.
		SetPreservesPitch(true).
		SetSpeed(p.speed).
		SetVolume(p.volume)
	stream.OnEnd(func() {
		p.trackEnded(stream)
	})
	return nil
}

// trackEnded advances to the next track when one finishes.
func (p *Playlist) trackEnded(ended *audio.Player) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed || p.player != ended {
		return
	}
	ended.Close()
	p.player = nil

	if p.index+1 >= len(p.tracks) {
		p.playing = false
		return
	}

	p.index++
	if err := p.openLocked(p.index); err != nil {
		p.err = err
		p.playing = false
		return
	}
	p.player.Play()
}

// indexAtLocked returns the track containing position. The caller holds p.mu.
func (p *Playlist) indexAtLocked(position time.Duration) int {
	if len(p.tracks) == 0 {
		return 0
	}
	index := sort.Search(len(p.starts), func(i int) bool {
		return p.starts[i] > position
	}) - 1
	if index < 0 {
		index = 0
	}
	if index >= len(p.tracks) {
		index = len(p.tracks) - 1
	}
	return index
}
