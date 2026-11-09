package player

import (
	"errors"
	"testing"
	"time"
)

func TestNewBuildsTimeline(t *testing.T) {
	playlist := New([]Track{
		{Path: "one.mp3", Duration: 10 * time.Second},
		{Path: "two.mp3", Duration: 20 * time.Second},
	})
	defer playlist.Close()

	if got := playlist.Duration(); got != 30*time.Second {
		t.Errorf("Duration() = %v, want 30s", got)
	}
	if got := playlist.Position(); got != 0 {
		t.Errorf("Position() = %v, want 0", got)
	}
	if got := playlist.Speed(); got != 1 {
		t.Errorf("Speed() = %v, want 1", got)
	}
	if playlist.Playing() {
		t.Error("Playing() = true, want false")
	}

	starts := []time.Duration{0, 10 * time.Second}
	for index, want := range starts {
		if got := playlist.starts[index]; got != want {
			t.Errorf("starts[%d] = %v, want %v", index, got, want)
		}
	}
}

func TestIndexAt(t *testing.T) {
	playlist := New([]Track{
		{Path: "one.mp3", Duration: 10 * time.Second},
		{Path: "two.mp3", Duration: 20 * time.Second},
	})
	defer playlist.Close()

	tests := []struct {
		position time.Duration
		want     int
	}{
		{0, 0},
		{5 * time.Second, 0},
		{10 * time.Second, 1},
		{29 * time.Second, 1},
		{time.Minute, 1},
	}
	for _, test := range tests {
		if got := playlist.indexAtLocked(test.position); got != test.want {
			t.Errorf("indexAtLocked(%v) = %d, want %d", test.position, got, test.want)
		}
	}
}

func TestEmptyPlaylist(t *testing.T) {
	playlist := New(nil)
	defer playlist.Close()

	if got := playlist.Duration(); got != 0 {
		t.Errorf("Duration() = %v, want 0", got)
	}
	if err := playlist.Play(); err == nil {
		t.Error("Play() on an empty playlist returned no error")
	}
	if got := playlist.Position(); got != 0 {
		t.Errorf("Position() = %v, want 0", got)
	}
}

func TestClosedPlaylist(t *testing.T) {
	playlist := New([]Track{{Path: "one.mp3", Duration: time.Second}})
	playlist.Close()

	if err := playlist.Play(); !errors.Is(err, ErrClosed) {
		t.Errorf("Play() after Close() = %v, want ErrClosed", err)
	}

	// Closing twice is safe.
	playlist.Close()
}

func TestSpeedAndVolume(t *testing.T) {
	playlist := New([]Track{{Path: "one.mp3", Duration: time.Second}})
	defer playlist.Close()

	playlist.SetSpeed(1.5)
	if got := playlist.Speed(); got != 1.5 {
		t.Errorf("Speed() = %v, want 1.5", got)
	}
	playlist.SetSpeed(0)
	if got := playlist.Speed(); got != 1.5 {
		t.Errorf("Speed() = %v after SetSpeed(0), want it unchanged", got)
	}

	playlist.SetVolume(2)
	if got := playlist.volume; got != 1 {
		t.Errorf("volume = %v after SetVolume(2), want 1", got)
	}
	playlist.SetVolume(-1)
	if got := playlist.volume; got != 0 {
		t.Errorf("volume = %v after SetVolume(-1), want 0", got)
	}
}
