package transcribe

import "time"

// Word is one spoken word and the part of the audiobook where it is heard.
type Word struct {
	Text  string        `json:"text"`
	Start time.Duration `json:"start"`
	End   time.Duration `json:"end"`
}

// File describes one audio file in a book's timeline.
type File struct {
	Path     string        `json:"path"`
	Size     int64         `json:"size"`
	ModTime  int64         `json:"modTime"`
	Duration time.Duration `json:"duration"`
	Start    time.Duration `json:"start"`
}

// Transcript is the timed text of a whole audiobook.
type Transcript struct {
	Version  int           `json:"version"`
	BookID   string        `json:"bookId"`
	Files    []File        `json:"files"`
	Words    []Word        `json:"words"`
	Duration time.Duration `json:"duration"`
}

// Progress reports a long job. Fraction is between 0 and 1, or negative when
// the job cannot estimate it.
type Progress struct {
	Stage    string
	Fraction float64
}
