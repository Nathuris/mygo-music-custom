package app

import (
	"fmt"
	"image"
	"testing"

	"github.com/egoist/mygo/ui"

	"mygo-music/internal/player"
)

// Shuffle and all three repeat modes remain usable as Now Playing's panes change.
func TestTransportModesAcrossPanes(t *testing.T) {
	a := newTestApp()
	a.pl.Cue(framesTracks(a))
	a.pl.SetLyrics(nil)
	a.nowPlaying, a.npPane = true, paneLyrics
	tt := ui.NewTester(a.view, 1200, 800)
	settled(tt)
	for _, pane := range []string{"Lyrics", "Cover", "Up Next", "Wall"} {
		if err := tt.Click(pane); err != nil {
			t.Fatal(err)
		}
		settled(tt)
		for _, want := range []bool{true, false} {
			if err := tt.Click("Shuffle"); err != nil {
				t.Fatal(err)
			}
			tt.Frame()
			if a.pl.Shuffle() != want {
				t.Fatalf("%s: shuffle is %v, want %v", pane, a.pl.Shuffle(), want)
			}
		}
		for _, want := range []player.RepeatMode{player.RepeatAll, player.RepeatOne, player.RepeatOff} {
			if err := tt.Click("Repeat"); err != nil {
				t.Fatal(err)
			}
			tt.Frame()
			if got := a.pl.Repeat(); got != want {
				t.Fatalf("%s: repeat is %v, want %v", pane, got, want)
			}
		}
	}
}

// With MM_FRAMES set, saves the transport modes over different cover colors.
func TestTransportModesFrames(t *testing.T) {
	dir := framesDir(t)
	for _, track := range []int{0, 1, 2} {
		for _, mode := range []string{"off", "all", "one"} {
			a := newTestApp()
			tracks := framesTracks(a)
			a.pl.Cue(tracks[track : track+1])
			a.pl.SetLyrics(nil)
			if mode != "off" {
				a.pl.ToggleShuffle()
				a.pl.CycleRepeat()
			}
			if mode == "one" {
				a.pl.CycleRepeat()
			}
			a.nowPlaying, a.npPane = true, paneLyrics
			tt := ui.NewTester(a.view, 1180, 780)
			settled(tt)
			left, ok := tt.Find("Shuffle")
			if !ok {
				t.Fatal("no shuffle button")
			}
			right, ok := tt.Find("Repeat")
			if !ok {
				t.Fatal("no repeat button")
			}
			img := tt.Image()
			scale := float32(img.Bounds().Dx()) / 1180
			rect := image.Rect(int((left.X-12)*scale), int((left.Y-8)*scale), int((right.X+right.W+12)*scale), int((right.Y+right.H+8)*scale))
			saveFrame(t, dir, fmt.Sprintf("transport-%d-%s.png", track, mode), img.SubImage(rect))
		}
	}
}
