package app

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"mygo-music/internal/lyrics"
	"mygo-music/internal/music"
)

// Lyrics leaving and returning ignore clicks while hidden, then seek again;
// hiding them previously removed the pane before its exit could finish.
func TestLyricsToggleRestoresLineSeeking(t *testing.T) {
	a := newTestApp()
	a.pl.Cue([]music.Track{song("1", "一")})
	a.pl.SetLyrics(lyrics.Build("[00:00.00]First line\n[00:20.00]Second line", ""))
	a.pl.Hold(5 * time.Second)
	a.nowPlaying, a.npPane = true, paneLyrics
	tt := ui.NewTester(a.view, 1200, 800)
	settled(tt)
	if err := tt.Click("Hide Lyrics"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if err := tt.Click("Second line"); err == nil {
		t.Fatal("departing lyrics accepted a click")
	}
	settled(tt)
	if err := tt.Click("Show Lyrics"); err != nil {
		t.Fatal(err)
	}
	settled(tt)
	if err := tt.Click("Second line"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if pos := a.pl.Position(); pos != 20*time.Second {
		t.Fatalf("click sought to %v, want 20s", pos)
	}
}

// Hiding lyrics centers the cover and controls, including for an instrumental,
// remembers the choice, and lets the listener restore lyrics and use playback.
func TestLyricsToggleCentersPlayback(t *testing.T) {
	for _, size := range []struct {
		name string
		w, h int
	}{{"wide", 1500, 900}, {"short", 880, 600}} {
		for _, hasLyrics := range []bool{false, true} {
			name := "instrumental"
			if hasLyrics {
				name = "lyrics"
			}
			t.Run(size.name+"/"+name, func(t *testing.T) {
				a := newTestApp()
				a.settings.path = filepath.Join(t.TempDir(), "settings.json")
				a.pl.Cue([]music.Track{song("1", "一"), song("2", "二")})
				a.pl.SetLyrics(nil)
				if hasLyrics {
					a.pl.SetLyrics(lyrics.Build("[00:00.00]First line\n[00:20.00]Second line", ""))
				}
				a.pl.Hold(5 * time.Second)
				a.nowPlaying, a.npPane = true, paneLyrics
				tt := ui.NewTester(a.view, size.w, size.h)
				settle := func() {
					tt.Frame()
					tt.Advance(time.Second)
					tt.Frame()
				}
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
					settle()
				}
				settle()
				cover, progress := find("Album Cover"), find("Progress")
				for range 2 {
					click("Hide Lyrics")
					if !a.settings.HideLyrics {
						t.Fatal("lyrics were not hidden")
					}
					for _, label := range []string{"Album Cover", "Progress", "Play/Pause"} {
						r := find(label)
						if math.Abs(float64(r.X+r.W/2)-float64(size.w)/2) > 1 {
							t.Errorf("%s is not centered: %+v", label, r)
						}
					}
					if _, ok := tt.Find("First line"); ok {
						t.Error("hidden lyrics still take input")
					}
					b, err := os.ReadFile(a.settings.path)
					if err != nil {
						t.Fatal(err)
					}
					var saved settings
					if err := json.Unmarshal(b, &saved); err != nil || !saved.HideLyrics {
						t.Fatalf("hidden lyrics were not saved: %s, %v", b, err)
					}
					click("Show Lyrics")
					if a.settings.HideLyrics {
						t.Fatal("lyrics were not restored")
					}
					if got := find("Album Cover"); got != cover {
						t.Errorf("cover did not return to the left: %+v, want %+v", got, cover)
					}
					if got := find("Progress"); got != progress {
						t.Errorf("progress did not return to the left: %+v, want %+v", got, progress)
					}
					if hasLyrics {
						find("First line")
					}
					if a.pl.Current().ID != "1" || a.pl.Position() != 5*time.Second {
						t.Fatal("toggling lyrics changed playback")
					}
				}
				click("Hide Lyrics")
				click("Previous")
				if a.pl.Current().ID != "2" || !a.settings.HideLyrics {
					t.Fatal("centered playback did not keep its layout while changing songs")
				}
			})
		}
	}
}

// Switching lyrics back on during the move keeps the controls together,
// lets them take clicks, and returns them to the left without a jump.
func TestLyricsToggleDuringMove(t *testing.T) {
	a := newTestApp()
	a.pl.Cue([]music.Track{song("1", "一"), song("2", "二")})
	a.pl.SetLyrics(nil)
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
	left, cover := find("Progress"), find("Album Cover")
	click("Hide Lyrics")
	tt.Advance(200 * time.Millisecond)
	tt.Frame()
	moving := find("Progress")
	if moving.X <= left.X || moving.X+moving.W/2 >= 600 {
		t.Fatalf("the controls jumped instead of moving: %+v", moving)
	}
	if got := find("Album Cover"); math.Abs(float64((got.X-cover.X)-(moving.X-left.X))) > 1 {
		t.Fatal("the cover moved separately from the controls")
	}
	click("Previous")
	if a.pl.Current().ID != "2" {
		t.Fatal("the moving controls did not take the click")
	}
	click("Show Lyrics")
	if got := find("Progress"); math.Abs(float64(got.X-moving.X)) > 1 {
		t.Fatalf("reversing the move jumped from %v to %v", moving.X, got.X)
	}
	settled(tt)
	if got := find("Progress"); got != left || a.settings.HideLyrics {
		t.Fatalf("the controls did not return to the left: %+v", got)
	}
}
