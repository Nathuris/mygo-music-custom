package app

import (
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"mygo-music/internal/lyrics"
	"mygo-music/internal/music"
)

// Entering and leaving the wall used to replace the cover and controls at once.
func TestWallTransitionsKeepPlaybackControls(t *testing.T) {
	for _, start := range []string{"Cover", "Lyrics", "Up Next"} {
		t.Run(start, func(t *testing.T) {
			a := newTestApp()
			a.pl.Cue(framesTracks(a))
			a.pl.SetLyrics(nil)
			a.nowPlaying, a.npPane = true, paneLyrics
			tt := ui.NewTester(a.view, 1200, 800)
			settled(tt)
			if err := tt.Click(start); err != nil {
				t.Fatal(err)
			}
			settled(tt)
			find := func(label string) ui.Rect {
				t.Helper()
				r, ok := tt.Find(label)
				if !ok {
					t.Fatalf("missing %s", label)
				}
				return r
			}
			near := func(a, b ui.Rect) bool {
				return math.Abs(float64(a.X-b.X))+math.Abs(float64(a.Y-b.Y))+
					math.Abs(float64(a.W-b.W))+math.Abs(float64(a.H-b.H)) < 3
			}
			cover, progress := find("Album Cover"), find("Progress")
			if err := tt.Click("Wall"); err != nil {
				t.Fatal(err)
			}
			tt.Frame()
			if got := find("Album Cover"); !near(got, cover) {
				t.Fatalf("cover jumped from %+v to %+v", cover, got)
			}
			if got := find("Progress"); !near(got, progress) {
				t.Fatalf("progress jumped from %+v to %+v", progress, got)
			}
			tt.Advance(140 * time.Millisecond)
			tt.Frame()
			moving := find("Album Cover")
			if near(moving, cover) {
				t.Fatal("the cover did not move toward the wall")
			}
			if err := tt.Click("Shuffle"); err != nil {
				t.Fatal(err)
			}
			if !a.pl.Shuffle() {
				t.Fatal("moving controls did not take input")
			}
			if err := tt.Click(start); err != nil {
				t.Fatal(err)
			}
			tt.Frame()
			if got := find("Album Cover"); !near(got, moving) {
				t.Fatalf("reversing the cover jumped from %+v to %+v", moving, got)
			}
			settled(tt)
			if got := find("Album Cover"); !near(got, cover) {
				t.Fatalf("cover returned to %+v, want %+v", got, cover)
			}
		})
	}
}

// A queue fading into the cover must not accept clicks or briefly reactivate full lyrics.
func TestQueueToCoverDisablesDepartingContent(t *testing.T) {
	a := newTestApp()
	a.pl.Cue([]music.Track{song("1", "First song"), song("2", "Second song")})
	a.pl.SetLyrics(lyrics.Build("[00:00.00]First line\n[00:20.00]Second line", ""))
	a.pl.Hold(5 * time.Second)
	a.nowPlaying, a.npPane = true, paneQueue
	tt := ui.NewTester(a.view, 1200, 800)
	settled(tt)
	if err := tt.Click("Cover"); err != nil {
		t.Fatal(err)
	}
	tt.Advance(80 * time.Millisecond)
	tt.Frame()
	if err := tt.Click("Second song"); err == nil {
		t.Fatal("the departing queue still accepted input")
	}
	if err := tt.Click("Second line"); err == nil {
		t.Fatal("full lyrics became interactive on the way to the cover")
	}
	if a.pl.Current().ID != "1" || a.pl.Position() != 5*time.Second {
		t.Fatal("departing content changed playback")
	}
	settled(tt)
	if err := tt.Click("Up Next"); err != nil {
		t.Fatal(err)
	}
	settled(tt)
	if err := tt.Click("Second song"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.pl.Current().ID != "2" {
		t.Fatal("the queue did not accept input when reopened")
	}
}

// The compact lyrics view follows playback and survives a visit to the queue.
func TestCompactLyricsFollowPlayback(t *testing.T) {
	a := newTestApp()
	a.pl.Cue([]music.Track{song("1", "一"), song("2", "二")})
	a.pl.SetLyrics(lyrics.Build("[00:00.00]First line\n[00:20.00]Second line", "[00:00.00]First translation\n[00:20.00]Second translation"))
	a.pl.Hold(5 * time.Second)
	a.nowPlaying, a.npPane = true, paneLyrics
	tt := ui.NewTester(a.view, 1200, 800)
	settled(tt)
	if err := tt.Click("Cover"); err != nil {
		t.Fatal(err)
	}
	settled(tt)
	for _, text := range []string{"First line", "First translation"} {
		if !tt.HasText(text) {
			t.Fatalf("compact view missing %q: %v", text, tt.Texts())
		}
	}
	a.pl.Hold(25 * time.Second)
	tt.Frame()
	for _, text := range []string{"Second line", "Second translation"} {
		if !tt.HasText(text) {
			t.Fatalf("compact view did not follow playback to %q: %v", text, tt.Texts())
		}
	}
	if err := tt.Click("Up Next"); err != nil {
		t.Fatal(err)
	}
	settled(tt)
	if err := tt.Click("Cover"); err != nil {
		t.Fatal(err)
	}
	settled(tt)
	if !a.settings.HideLyrics || !tt.HasText("Second translation") {
		t.Fatal("returning from the queue lost the compact view")
	}
	if err := tt.Click("Lyrics"); err != nil {
		t.Fatal(err)
	}
	settled(tt)
	if a.settings.HideLyrics {
		t.Fatal("the full lyrics were not restored")
	}
}

// Credits stay fixed when lyrics appear, change length or disappear below them.
func TestCoverCreditsStayPutWithLyrics(t *testing.T) {
	a := newTestApp()
	a.pl.Cue([]music.Track{song("1", "Cherry Candy", "Lunak")})
	a.pl.SetLyrics(lyrics.Build("[00:10.00]First line\n[00:20.00]A much longer line that changes the lyrics below the credits", "[00:10.00]Translation"))
	a.pl.Hold(5 * time.Second)
	a.nowPlaying, a.npPane = true, paneCover
	tt := ui.NewTester(a.view, 1180, 780)
	settled(tt)
	find := func(label string) ui.Rect {
		t.Helper()
		r, ok := tt.Find(label)
		if !ok {
			t.Fatalf("missing %s", label)
		}
		return r
	}
	title, artist := find("Cherry Candy"), find("Lunak")
	for _, pos := range []time.Duration{11 * time.Second, 21 * time.Second, 5 * time.Second} {
		a.pl.Hold(pos)
		for range 7 {
			tt.Frame()
			if got := find("Cherry Candy"); math.Abs(float64(got.Y-title.Y)) > 1 {
				t.Fatalf("at %v the title moved from %v to %v", pos, title.Y, got.Y)
			}
			if got := find("Lunak"); math.Abs(float64(got.Y-artist.Y)) > 1 {
				t.Fatalf("at %v the artist moved from %v to %v", pos, artist.Y, got.Y)
			}
			tt.Advance(80 * time.Millisecond)
		}
	}
}

// With MM_FRAMES set, saves the transition between full and compact lyrics.
func TestPaneTransitionFrames(t *testing.T) {
	dir := framesDir(t)
	a := newTestApp()
	a.pl.Cue(framesTracks(a))
	a.pl.SetLyrics(lyrics.Build("[00:00.00]First line\n[00:20.00]Second line", "[00:00.00]First translation"))
	a.pl.Hold(5 * time.Second)
	a.nowPlaying, a.npPane = true, paneLyrics
	tt := ui.NewTester(a.view, 1180, 780)
	settled(tt)
	for step, tab := range []string{"Cover", "Lyrics", "Up Next", "Lyrics"} {
		if err := tt.Click(tab); err != nil {
			t.Fatal(err)
		}
		for i := range 9 {
			tt.Frame()
			saveFrame(t, dir, fmt.Sprintf("transition-%d-%d.png", step, i), tt.Image())
			tt.Advance(60 * time.Millisecond)
		}
		settled(tt)
	}
}

// With MM_FRAMES set, saves transitions between the queue, cover and wall.
func TestQueueWallTransitionFrames(t *testing.T) {
	dir := framesDir(t)
	a := newTestApp()
	tracks := framesTracks(a)
	a.pl.Cue(tracks)
	a.pl.SetLyrics(lyrics.Build("[00:00.00]First line\n[00:20.00]Second line", "[00:00.00]First translation"))
	a.pl.Hold(5 * time.Second)
	a.nowPlaying, a.npPane = true, paneLyrics
	tt := ui.NewTester(a.view, 1180, 780)
	settled(tt)
	for step, tab := range []string{"Up Next", "Cover", "Wall", "Lyrics", "Wall", "Up Next", "Wall", "Cover"} {
		if err := tt.Click(tab); err != nil {
			t.Fatal(err)
		}
		for i := range 10 {
			tt.Frame()
			saveFrame(t, dir, fmt.Sprintf("queue-wall-%d-%d.png", step, i), tt.Image())
			tt.Advance(50 * time.Millisecond)
		}
		settled(tt)
	}
}
