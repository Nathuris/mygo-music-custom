package app

import (
	"fmt"
	"image"
	"os"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"mygo-music/internal/local"
	"mygo-music/internal/lyrics"
	"mygo-music/internal/music"
)

// framesDir is where MM_FRAMES says to save a look at the views, which a test
// skips without.
func framesDir(t *testing.T) string {
	t.Helper()
	dir := os.Getenv("MM_FRAMES")
	if dir == "" {
		t.Skip()
	}
	return dir
}

// With MM_FRAMES set, saves Home as it greets a listener with no music yet.
func TestHomeEmptyFrame(t *testing.T) {
	dir := framesDir(t)
	a := localApp(t)
	tt := ui.NewTester(a.view, 1180, 780)
	settled(tt)
	saveFrame(t, dir, "home-empty.png", tt.Image())
}

// framesTracks are songs with covers, by an invented band.
func framesTracks(a *app) []music.Track {
	covers := []struct{ from, to ui.Color }{
		{ui.RGB(40, 90, 170), ui.RGB(220, 120, 160)},
		{ui.RGB(230, 150, 70), ui.RGB(120, 40, 90)},
		{ui.RGB(30, 130, 110), ui.RGB(210, 220, 140)},
	}
	band := music.Artist{ID: "1", Name: "The Polar Lights"}
	var tracks []music.Track
	for i, name := range []string{"Northern Lights", "Slow Tide", "A Long Way North", "Paper Boats", "The Harbor at Night", "Quiet Hours", "Last Ferry Home", "Snowfield"} {
		url := fmt.Sprintf("cover:%d", i%len(covers))
		giveCover(a, url, coverImage(covers[i%len(covers)].from, covers[i%len(covers)].to))
		tracks = append(tracks, music.Track{
			ID: fmt.Sprint(i + 1), Source: "fake", Name: name, Artists: []music.Artist{band},
			Album:    music.Album{ID: fmt.Sprint(10 + i%len(covers)), Name: "Polar Night", ImageURL: url},
			Duration: int64(180000 + 21000*i),
		})
	}
	return tracks
}

// With MM_FRAMES set, saves a playlist's page.
func TestPlaylistFrame(t *testing.T) {
	dir := framesDir(t)
	a := newTestApp()
	a.account = &music.Account{ID: "1", Name: "Alex"}
	tracks := framesTracks(a)
	a.mine = []music.Playlist{
		{ID: "10", Name: "Evening Walks", ImageURL: "cover:0", Mine: true},
		{ID: "11", Name: "Focus", ImageURL: "cover:1", Mine: true},
	}
	a.saved = []music.Playlist{{ID: "20", Name: "Staff Picks", ImageURL: "cover:2", Saved: true}}
	a.playlists["10"] = &music.Playlist{ID: "10", Name: "Evening Walks", Owner: "Alex", Description: "For the long way home.", ImageURL: "cover:0", Mine: true, Tracks: tracks}
	a.route = route{page: pagePlaylist, src: "fake", id: "10"}
	tt := ui.NewTester(a.view, 1180, 780)
	settled(tt)
	saveFrame(t, dir, "playlist.png", tt.Image())
}

// With MM_FRAMES set, saves Now Playing with lyrics following a song.
func TestNowPlayingFrame(t *testing.T) {
	dir := framesDir(t)
	a := newTestApp()
	a.account = &music.Account{ID: "1", Name: "Alex"}
	tracks := framesTracks(a)
	a.pl.Cue(tracks)
	a.pl.SetLyrics(lyrics.Build(
		"[00:04.00]The sky goes green at the edges\n[00:09.00]And everyone stops to look\n[00:14.00]Nobody says a word\n[00:19.00]Nobody needs to\n[00:24.00]We stand out on the ice\n[00:29.00]Until the light goes home",
		"",
	))
	a.pl.Hold(10 * time.Second)
	a.nowPlaying, a.npPane = true, paneLyrics
	tt := ui.NewTester(a.view, 1180, 780)
	tt.SetDark(true)
	tt.Advance(2 * time.Second)
	tt.Frame()
	saveFrame(t, dir, "now-playing.png", tt.Image())
	motion := func(name string) {
		var shots []image.Image
		for range 12 {
			shots = append(shots, tt.Image())
			tt.Advance(60 * time.Millisecond)
			tt.Frame()
		}
		saveFrame(t, dir, name, contactSheet(shots, 4))
	}
	if err := tt.Click("Cover"); err != nil {
		t.Fatal(err)
	}
	motion("now-playing-hide-lyrics-motion.png")
	settled(tt)
	saveFrame(t, dir, "now-playing-centered.png", tt.Image())
	a.pl.SetLyrics(nil)
	tt.Frame()
	saveFrame(t, dir, "now-playing-instrumental-centered.png", tt.Image())
	if err := tt.Click("Lyrics"); err != nil {
		t.Fatal(err)
	}
	motion("now-playing-show-lyrics-motion.png")
	settled(tt)
	saveFrame(t, dir, "now-playing-instrumental-lyrics.png", tt.Image())
}

// With MM_FRAMES set, saves the cover's song credits and lyrics at different window sizes.
func TestCoverCreditsFrames(t *testing.T) {
	dir := framesDir(t)
	for _, height := range []int{580, 780, 1000} {
		a := newTestApp()
		tracks := framesTracks(a)
		tracks[0].Artists = append(tracks[0].Artists, music.Artist{Name: "Another Artist"})
		a.pl.Cue(tracks)
		a.pl.SetLyrics(lyrics.Build("[00:00.00]The sky goes green at the edges", "[00:00.00]天边泛起了绿光"))
		a.pl.Hold(5 * time.Second)
		a.nowPlaying, a.npPane = true, paneCover
		tt := ui.NewTester(a.view, 1180, height)
		settled(tt)
		saveFrame(t, dir, fmt.Sprintf("cover-credits-%d.png", height), tt.Image())
	}
	a := newTestApp()
	tracks := framesTracks(a)
	tracks[0].Name = "A Very Long Song Name That Keeps Going and Going and Going and Going and Going"
	tracks[0].Artists = []music.Artist{{Name: "An Extremely Long Artist Name"}, {Name: "Another Very Long Artist Name"}}
	a.pl.Cue(tracks)
	a.pl.SetLyrics(lyrics.Build("[00:00.00]A long line that goes on and on as the sky turns green and everyone stops and looks at the stars above the horizon tonight", "[00:00.00]一条很长的翻译，讲述人们停下来欣赏天空和地平线上的星光"))
	a.pl.Hold(5 * time.Second)
	a.nowPlaying, a.npPane = true, paneCover
	tt := ui.NewTester(a.view, 880, 580)
	settled(tt)
	saveFrame(t, dir, "cover-credits-long.png", tt.Image())
}

// With MM_FRAMES set, saves the cover credits as a lyric appears and leaves.
func TestCoverCreditsMotionFrames(t *testing.T) {
	dir := framesDir(t)
	for _, height := range []int{580, 780} {
		a := newTestApp()
		a.pl.Cue(framesTracks(a))
		a.pl.SetLyrics(lyrics.Build("[00:10.00]The sky goes green at the edges", "[00:10.00]天边泛起了绿光"))
		a.pl.Hold(5 * time.Second)
		a.nowPlaying, a.npPane = true, paneCover
		tt := ui.NewTester(a.view, 1180, height)
		settled(tt)
		for step, pos := range []time.Duration{11 * time.Second, 5 * time.Second} {
			a.pl.Hold(pos)
			for i := range 9 {
				tt.Frame()
				saveFrame(t, dir, fmt.Sprintf("cover-credits-motion-%d-%d-%d.png", height, step, i), tt.Image())
				tt.Advance(60 * time.Millisecond)
			}
			settled(tt)
		}
	}
}

// With MM_FRAMES set, saves Now Playing for a local song with no cover.
func TestNowPlayingNoCoverFrame(t *testing.T) {
	dir := framesDir(t)
	a := newTestApp()
	tracks := framesTracks(a)
	for i := range tracks {
		tracks[i].Album.ImageURL = ""
	}
	a.pl.Cue(tracks)
	a.pl.Hold(10 * time.Second)
	a.nowPlaying, a.npPane = true, paneQueue
	tt := ui.NewTester(a.view, 1180, 780)
	tt.SetDark(true)
	tt.Advance(2 * time.Second)
	tt.Frame()
	saveFrame(t, dir, "now-playing-no-cover.png", tt.Image())
}

// With MM_FRAMES set, saves Now Playing growing out of the player bar's
// cover, a frame every 50 ms, and shrinking back into it, in a sheet each.
func TestNowPlayingMotionFrames(t *testing.T) {
	dir := framesDir(t)
	a := newTestApp()
	a.pl.Cue(framesTracks(a))
	a.pl.SetLyrics(nil)
	a.pl.Hold(10 * time.Second)
	a.npPane = paneQueue
	tt := ui.NewTester(a.view, 1180, 780)
	tt.SetDark(true)
	settled(tt)
	sheet := func(name string) {
		var shots []image.Image
		for range 12 {
			shots = append(shots, tt.Image())
			tt.Advance(50 * time.Millisecond)
		}
		saveFrame(t, dir, name, contactSheet(shots, 4))
	}
	if err := tt.Click("Open Now Playing"); err != nil {
		t.Fatal(err)
	}
	sheet("np-open.png")
	settled(tt)
	a.nowPlaying = false
	tt.Frame()
	sheet("np-close.png")

	// Pulled down halfway, then let go to fall into the bar.
	a.nowPlaying = true
	settled(tt)
	tt.Press(450, 300)
	tt.Frame()
	for i := 1; i <= 10; i++ {
		tt.Advance(16 * time.Millisecond)
		tt.Move(450, 300+float32(i)*24)
		tt.Frame()
	}
	saveFrame(t, dir, "np-pulled.png", tt.Image())
	tt.Release(450, 540)
	tt.Frame()
	sheet("np-let-go.png")
}

// contactSheet lays shots out in rows of cols, each a third of its size.
func contactSheet(shots []image.Image, cols int) image.Image {
	b := shots[0].Bounds()
	cw, ch := b.Dx()/3, b.Dy()/3
	rows := (len(shots) + cols - 1) / cols
	out := image.NewRGBA(image.Rect(0, 0, cw*cols, ch*rows))
	for i, shot := range shots {
		x0, y0 := (i%cols)*cw, (i/cols)*ch
		for y := range ch {
			for x := range cw {
				out.Set(x0+x, y0+y, shot.At(x*3, y*3))
			}
		}
	}
	return out
}

// With MM_FRAMES set, saves the settings as they list the sources: the local
// files and a plugin that asks for a couple of things.
func TestSettingsFrame(t *testing.T) {
	dir := framesDir(t)
	a := newTestApp()
	a.libs = nil
	a.local = local.New(t.TempDir())
	addFake(a, a.local)
	plugin := newFakeSource("radio-garden")
	plugin.info.Name = "Example Source"
	plugin.info.Version = "1.2.0"
	plugin.info.Settings = []music.Setting{
		{Key: "apiKey", Label: "API Key", Help: "Keys are free for personal use.", Link: "https://example.com/keys"},
		{Key: "region", Label: "Region"},
	}
	plugin.vals["region"] = "Nordics"
	a.library = addFake(a, plugin)
	a.account = &music.Account{ID: "1", Name: "Alex"}
	a.settingsPage = "plugins"
	tt := ui.NewTester(a.settingsView, 780, 720)
	settled(tt)
	saveFrame(t, dir, "settings.png", tt.Image())
}
