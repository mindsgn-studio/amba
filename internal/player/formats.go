package player

import (
	// Register the audio formats the player can decode. WAV is built in.
	_ "github.com/egoist/mygo/audio/mp3"
	_ "github.com/egoist/mygo/audio/vorbis"
)
