package app

import (
	"fmt"
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
	// At full volume the percent sign previously wrapped onto another line.
	tt.Press(r.X+r.W*0.8, y)
	tt.Move(r.X+r.W+10, y)
	tt.Release(r.X+r.W+10, y)
	tt.Frame()
	if a.volume < 0.99 || !tt.HasText("100%") {
		t.Fatalf("volume at maximum: %v, texts: %v", a.volume, tt.Texts())
	}
}

// Switching to the cover used to hide the volume control, making it impossible to adjust.
func TestCoverVolumeRemainsAvailable(t *testing.T) {
	for _, width := range []int{880, 1180, 1500} {
		t.Run(fmt.Sprint(width), func(t *testing.T) {
			a := newTestApp()
			a.pl.Cue(framesTracks(a))
			a.nowPlaying, a.npPane = true, paneLyrics
			tt := ui.NewTester(a.view, width, 780)
			settled(tt)
			if err := tt.Click("Cover"); err != nil {
				t.Fatal(err)
			}
			settled(tt)
			r, ok := tt.Find("Volume")
			if !ok {
				t.Fatal("no volume slider on the cover")
			}
			x, y := r.X+r.W*0.7, r.Y+r.H/2
			tt.Press(x, y)
			tt.Release(x, y)
			tt.Frame()
			if math.Abs(a.volume-0.7) > 0.02 {
				t.Fatalf("cover slider set volume to %v", a.volume)
			}
			if err := tt.Click("Mute"); err != nil {
				t.Fatal(err)
			}
			tt.Frame()
			if a.volume != 0 {
				t.Fatalf("cover mute left volume at %v", a.volume)
			}
			if err := tt.Click("Mute"); err != nil {
				t.Fatal(err)
			}
			tt.Frame()
			if math.Abs(a.volume-0.7) > 0.02 {
				t.Fatalf("cover unmute restored volume to %v", a.volume)
			}
			if err := tt.Click("Lyrics"); err != nil {
				t.Fatal(err)
			}
			settled(tt)
			if _, ok := tt.Find("Volume"); !ok || math.Abs(a.volume-0.7) > 0.02 {
				t.Fatal("switching pages lost the volume control or its value")
			}
		})
	}
}

// The song picker used to fade out and disable the volume control entirely.
func TestWallVolumeRemainsAvailable(t *testing.T) {
	for _, width := range []int{880, 1180, 1500} {
		t.Run(fmt.Sprint(width), func(t *testing.T) {
			a := newTestApp()
			a.pl.Cue(framesTracks(a))
			a.nowPlaying, a.npPane = true, paneLyrics
			tt := ui.NewTester(a.view, width, 780)
			settled(tt)
			if err := tt.Click("Wall"); err != nil {
				t.Fatal(err)
			}
			settled(tt)
			r, ok := tt.Find("Volume")
			if !ok {
				t.Fatal("no volume slider on the song picker")
			}
			x, y := r.X+r.W*0.7, r.Y+r.H/2
			tt.Press(x, y)
			tt.Release(x, y)
			tt.Frame()
			if math.Abs(a.volume-0.7) > 0.02 {
				t.Fatalf("song picker slider set volume to %v", a.volume)
			}
			if err := tt.Click("Mute"); err != nil {
				t.Fatal(err)
			}
			tt.Frame()
			if a.volume != 0 {
				t.Fatalf("song picker mute left volume at %v", a.volume)
			}
			if err := tt.Click("Mute"); err != nil {
				t.Fatal(err)
			}
			tt.Frame()
			if math.Abs(a.volume-0.7) > 0.02 {
				t.Fatalf("song picker unmute restored volume to %v", a.volume)
			}
			if err := tt.Click("Cover"); err != nil {
				t.Fatal(err)
			}
			settled(tt)
			if _, ok := tt.Find("Volume"); !ok || math.Abs(a.volume-0.7) > 0.02 {
				t.Fatal("switching pages lost the volume control or its value")
			}
		})
	}
}

// With MM_FRAMES set, saves both lyrics layouts at several volume levels.
func TestNowPlayingVolumeFrames(t *testing.T) {
	dir := framesDir(t)
	for _, hidden := range []bool{false, true} {
		for _, volume := range []float64{0, 0.5, 1} {
			a := newTestApp()
			a.pl.Cue(framesTracks(a))
			a.settings.HideLyrics = hidden
			a.nowPlaying, a.npPane, a.volume = true, paneLyrics, volume
			if hidden {
				a.npPane = paneCover
			}
			tt := ui.NewTester(a.view, 1180, 780)
			settled(tt)
			saveFrame(t, dir, fmt.Sprintf("volume-%v-%d.png", hidden, int(volume*100)), tt.Image())
		}
	}
}
