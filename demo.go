// Demo usage of the go-sound Sounds library.
package main

import (
	"fmt"
	"runtime"

	"github.com/padster/go-sound/output"
	s "github.com/padster/go-sound/sounds"
)

func main() {
	// TODO - fix this here?
	runtime.GOMAXPROCS(4)

	fmt.Println("Building sound...")

	// TODO - fix it so it works at lower values (e.g. 215)
	mspb := 404 // ~= 148 bmp ~= 400 ms/b
	b := float64(mspb * 4)
	// var chordC s.Sound = s.ConcatSounds(
	// s.NewTimedSound(s.NewSineWave(523.25), b), // C
	// s.NewTimedSound(s.NewSineWave(659.25), b), // E
	// s.NewTimedSound(s.NewSineWave(783.99), b), // G
	// )
	// chordC = s.NewADSREnvelope(chordC, 50, 250, 0.3, 100)
	// chordA := s.SumSounds(
	// s.NewSineWave(440), s.NewSineWave(523.25), s.NewSineWave(659.25))
	// sound := s.ConcatSounds(
	// chordC,
	// s.NewTimedSound(chordA, b * 2),
	// )

	// sound := s.ConcatSounds(
	// s.NewADSREnvelope(
	// s.NewTimedSound(s.NewSineWave(523.25), b), 25, 200, 0.3, 100),
	// s.NewADSREnvelope(
	// s.NewTimedSound(s.NewSineWave(659.25), b), 25, 200, 0.3, 100),
	// s.NewADSREnvelope(
	// s.NewTimedSound(s.NewSineWave(783.99), b), 25, 200, 0.3, 100),
	// )

	// base := s.NewADSREnvelope(
	// s.NewTimedSound(s.NewSineWave(440), b), 25, 200, 0.3, 100)
	// sound := s.RepeatSound(chordC, 2)
	// sound := s.ConcatSounds(base,
	// s.NewADSREnvelope(s.NewTimedSound(s.NewSineWave(783.99), b), 25, 200, 0.3, 100))

	// chord := s.SumSounds(
	// s.LoadWavAsSound("piano.wav", 0),
	// s.NewSineWave(392.00),
	// )

	// sound := s.RepeatSound(s.NewTimedSound(chord, b), 3)

	// sound := s.RepeatSound(s.ConcatSounds(
	// s.NewADSREnvelope(s.NewTimedSound(s.ParseNotesToChord("CEbG", 4), b), 20, 100, 0.9, 20),
	// s.NewADSREnvelope(s.NewTimedSound(s.ParseNotesToChord("CEG", 4), b), 20, 100, 0.4, 20),
	// ), 4)

	// Hotel #1
	/*
		sound := s.ConcatSounds(
			s.NewTimedSound(s.ParseChord("Bm", 3), b),
			s.NewTimedSound(s.ParseChord("F#", 3), b),
			s.NewTimedSound(s.ParseChord("A", 3), b),
			s.NewTimedSound(s.ParseChord("E", 3), b),
			s.NewTimedSound(s.ParseChord("G", 3), b),
			s.NewTimedSound(s.ParseChord("D", 3), b),
			s.NewTimedSound(s.ParseChord("Em", 3), b),
			s.NewTimedSound(s.ParseChord("F#", 3), b),
		)
	*/

	// Hotel #2
	sound := s.ConcatSounds(
		s.NewADSREnvelope(s.NewTimedSound(s.GuitarChord("224432"), b), 50, 250, 0.8, 50),
		s.NewADSREnvelope(s.NewTimedSound(s.GuitarChord("244322"), b), 50, 250, 0.8, 50),
		s.NewADSREnvelope(s.NewTimedSound(s.GuitarChord("x02220"), b), 50, 250, 0.8, 50),
		s.NewADSREnvelope(s.NewTimedSound(s.GuitarChord("022100"), b), 50, 250, 0.8, 50),
		s.NewADSREnvelope(s.NewTimedSound(s.GuitarChord("320003"), b), 50, 250, 0.8, 50),
		s.NewADSREnvelope(s.NewTimedSound(s.GuitarChord("xx0232"), b), 50, 250, 0.8, 50),
		s.NewADSREnvelope(s.NewTimedSound(s.GuitarChord("022000"), b), 50, 250, 0.8, 50),
		s.NewADSREnvelope(s.NewTimedSound(s.GuitarChord("244322"), b), 50, 250, 0.8, 50),
	)

	// fmt.Printf("Playing sound.\n")
	// output.Play(s.RepeatSound(sound, 4)) // TODO - figure out why -1 doesn't work here
	output.WriteSoundToWav(sound, "hotcal.wav")

	// renderer := output.NewScreen(1500, 400)
	// renderer.Render(sound)

	// TODO - modem faker: 440 for 0.5s, pause for 1s, 440 for 0.5s, pause for 1s, 880 for 1.5s
}
