package app

import (
	"math"
	"testing"

	"github.com/egoist/mygo/ui"
)

// The displayed volume follows slider input and the mute button restores it.
func TestNowPlayingVolumeValueFollowsInput(t *testing.T) {
	a := newTestApp()
	a.pl.Cue(framesTracks(a))
	a.nowPlaying, a.npPane = true, paneLyrics
	tt := ui.NewTester(a.view, 1200, 800)
	settled(tt)
	r, ok := tt.Find("Volume")
	if !ok {
		t.Fatal("no volume slider")
	}
	x, y := r.X+r.W*0.8, r.Y+r.H/2
	tt.Press(x, y)
	tt.Release(x, y)
	tt.Frame()
	if math.Abs(a.volume-0.8) > 0.01 || !tt.HasText("80%") {
		t.Fatalf("volume after input: %v, texts: %v", a.volume, tt.Texts())
	}
	if err := tt.Click("Mute"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.volume != 0 || !tt.HasText("0%") {
		t.Fatalf("volume after muting: %v, texts: %v", a.volume, tt.Texts())
	}
	if err := tt.Click("Mute"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if math.Abs(a.volume-0.8) > 0.01 || !tt.HasText("80%") {
		t.Fatalf("volume after unmuting: %v, texts: %v", a.volume, tt.Texts())
	}
}
