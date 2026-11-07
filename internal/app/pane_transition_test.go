package app

import (
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// Returning from the stage keeps the cover still: matched controls used to
// change the column's height while they moved, making the cover jump.
func TestCoverStaysStillDuringPaneTransition(t *testing.T) {
	for _, height := range []int{580, 800, 1000} {
		for _, hidden := range []bool{false, true} {
			a := newTestApp()
			tracks := framesTracks(a)
			tracks[0].Name = "凝结的幻影"
			tracks[0].Artists[0].Name = "原色龙"
			tracks[0].Album.Name = "你好兽聚·深蓝遗章原声带"
			a.pl.Cue(tracks)
			a.pl.SetLyrics(nil)
			a.settings.HideLyrics = hidden
			a.nowPlaying = true
			tt := ui.NewTester(a.view, 1200, height)
			settled(tt)
			cover, ok := tt.Find("Album Cover")
			if !ok {
				t.Fatal("no cover")
			}
			if err := tt.Click("Stage"); err != nil {
				t.Fatal(err)
			}
			settled(tt)
			if err := tt.Click("Lyrics"); err != nil {
				t.Fatal(err)
			}
			for range 12 {
				tt.Frame()
				got, ok := tt.Find("Album Cover")
				if !ok || math.Abs(float64(got.Y-cover.Y)) > 1 {
					t.Fatalf("hidden %v: cover moved from %v to %v", hidden, cover.Y, got.Y)
				}
				tt.Advance(40 * time.Millisecond)
			}
		}
	}
}

// Changing panes keeps playback clickable as its progress bar moves from the
// actual lyrics layout, even mid-toggle, and reverses from where it reached.
func TestPaneTransitionKeepsPlayback(t *testing.T) {
	for _, mode := range []string{"lyrics", "centered", "centering"} {
		t.Run(mode, func(t *testing.T) {
			a := newTestApp()
			a.pl.Cue(framesTracks(a))
			a.pl.SetLyrics(nil)
			a.pl.Hold(10 * time.Second)
			a.settings.HideLyrics = mode == "centered"
			a.nowPlaying, a.npPane = true, paneLyrics
			tt := ui.NewTester(a.view, 1200, 800)
			settled(tt)
			find := func(label string) ui.Rect {
				t.Helper()
				r, ok := tt.Find(label)
				if !ok {
					t.Fatalf("missing %s", label)
				}
				return r
			}
			click := func(label string) {
				t.Helper()
				if err := tt.Click(label); err != nil {
					t.Fatal(err)
				}
				tt.Frame()
			}
			near := func(got, want ui.Rect) bool {
				return math.Abs(float64(got.X-want.X))+math.Abs(float64(got.Y-want.Y))+
					math.Abs(float64(got.W-want.W))+math.Abs(float64(got.H-want.H)) < 2
			}
			if mode == "centering" {
				click("Hide Lyrics")
				tt.Advance(100 * time.Millisecond)
				tt.Frame()
			}
			start, play := find("Progress"), find("Play/Pause")
			click("Stage")
			if got := find("Progress"); !near(got, start) {
				t.Fatalf("entering the stage jumped from %+v to %+v", start, got)
			}
			tt.Advance(140 * time.Millisecond)
			tt.Frame()
			moving := find("Progress")
			if moving.Y <= start.Y || moving.W <= start.W || moving.W >= 1104 {
				t.Fatalf("progress did not move down and grow: %+v", moving)
			}
			if got := find("Play/Pause"); got.Y <= play.Y {
				t.Fatalf("playback did not move down: %+v", got)
			}
			click("Shuffle")
			if !a.pl.Shuffle() {
				t.Fatal("the moving controls did not take the click")
			}
			click("Shuffle")
			click("Lyrics")
			if got := find("Progress"); !near(got, moving) {
				t.Fatalf("reversing the transition jumped from %+v to %+v", moving, got)
			}
			settled(tt)
			restored := find("Progress")
			if mode == "centering" {
				if math.Abs(float64(restored.X+restored.W/2-600)) > 1 {
					t.Fatalf("lyrics did not return to the centered layout: %+v", restored)
				}
			} else if !near(restored, start) {
				t.Fatalf("lyrics returned to %+v, want %+v", restored, start)
			}
			click("Stage")
			settled(tt)
			if got := find("Progress"); got.W != 1104 || got.Y <= start.Y {
				t.Fatalf("the stage's progress did not reach its place: %+v", got)
			}
			if _, ok := tt.Find("Volume"); ok {
				t.Error("the stage still has a volume slider")
			}
			click("Up Next")
			tt.Advance(100 * time.Millisecond)
			tt.Frame()
			click("Next")
			if a.pl.Current().ID != "2" {
				t.Fatal("the controls did not work while the queue faded in")
			}
			click("Lyrics")
			settled(tt)
			if a.npPane != paneLyrics || a.settings.HideLyrics != (mode != "lyrics") {
				t.Fatal("switching panes lost the lyrics layout choice")
			}
		})
	}
}

// With MM_FRAMES set, saves the separate content fades and controls' flight
// between the stage and each lyrics layout, and through the queue.
func TestPaneTransitionFrames(t *testing.T) {
	dir := framesDir(t)
	for _, mode := range []string{"lyrics", "centered"} {
		a := newTestApp()
		a.pl.Cue(framesTracks(a))
		a.pl.SetLyrics(nil)
		a.pl.Hold(10 * time.Second)
		a.settings.HideLyrics = mode == "centered"
		a.nowPlaying, a.npPane = true, paneLyrics
		tt := ui.NewTester(a.view, 1180, 780)
		settled(tt)
		for step, tab := range []string{"Stage", "Lyrics", "Up Next", "Stage"} {
			if err := tt.Click(tab); err != nil {
				t.Fatal(err)
			}
			for i := range 9 {
				tt.Frame()
				saveFrame(t, dir, fmt.Sprintf("%s-%d-to-%s-%d.png", mode, step, tab, i), tt.Image())
				tt.Advance(60 * time.Millisecond)
			}
			settled(tt)
		}
	}
}
