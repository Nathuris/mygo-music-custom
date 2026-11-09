package app

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"mygo-music/internal/music"
)

func diskTestJPEG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 800, 800))
	for y := range 800 {
		for x := range 800 {
			img.SetRGBA(x, y, color.RGBA{R: uint8(x), G: uint8(y), B: 90, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// A previously loaded network cover is available after restarting, without
// fetching it again, and the saved file is the resized thumbnail.
func TestArtDiskCacheSurvivesRestart(t *testing.T) {
	data := diskTestJPEG(t)
	var requests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		_, _ = w.Write(data)
	}))
	defer srv.Close()
	dir := t.TempDir()
	key := sized(srv.URL+"/cover.jpg", 96)
	largeKey := sized(srv.URL+"/cover.jpg", 600)
	posts := make(chan func(), 4)
	first := newArtCache(func(fn func()) { posts <- fn })
	first.disk = newArtDiskCache(dir)
	first.get(key, false)
	first.get(largeKey, false)
	runPost(t, posts)
	runPost(t, posts)
	if a := first.get(key, false); a == nil || a.failed {
		t.Fatal("the cover did not load")
	}
	diskKey, ok := artDiskKey(key)
	if !ok {
		t.Fatal("the network cover did not get a disk key")
	}
	cached := first.disk.read(diskKey)
	cfg, _, err := image.DecodeConfig(bytes.NewReader(cached))
	if err != nil || cfg.Width != 96 || cfg.Height != 96 {
		t.Fatalf("saved thumbnail is %dx%d: %v", cfg.Width, cfg.Height, err)
	}
	srv.Close()
	second := newArtCache(func(fn func()) { posts <- fn })
	second.disk = newArtDiskCache(dir)
	second.get(key, false)
	second.get(largeKey, false)
	runPost(t, posts)
	runPost(t, posts)
	if a := second.get(key, false); a == nil || a.failed || a.bmp == nil {
		t.Fatal("the cover did not load from disk while offline")
	}
	if a := second.get(largeKey, false); a == nil || a.failed || a.bmp == nil {
		t.Fatal("the larger cover did not load from disk while offline")
	}
	if got := requests.Load(); got != 1 {
		t.Fatalf("the cover was fetched %d times", got)
	}
	old := time.Now().Add(-artDiskAge - time.Hour)
	if err := os.Chtimes(first.disk.path(diskKey), old, old); err != nil {
		t.Fatal(err)
	}
	third := newArtCache(func(fn func()) { posts <- fn })
	third.disk = newArtDiskCache(dir)
	third.get(key, false)
	runPost(t, posts)
	if a := third.get(key, false); a == nil || a.failed || a.bmp == nil {
		t.Fatal("an expired cover was not available while offline")
	}
}

// Concurrent cover workers keep complete files and respect the shared limit.
func TestArtDiskCacheConcurrentWrites(t *testing.T) {
	d := newArtDiskCache(t.TempDir())
	d.limit = 4000
	var wg sync.WaitGroup
	for i := range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			key := strings.Repeat(string(rune('a'+i)), 64)
			d.put(key, bytes.Repeat([]byte{byte(i)}, 500))
		}()
	}
	wg.Wait()
	if d.size > d.limit {
		t.Fatalf("concurrent writes used %d bytes, limit %d", d.size, d.limit)
	}
	for key, entry := range d.files {
		if got := d.read(key); int64(len(got)) != entry.size {
			t.Fatalf("cached file has %d bytes, want %d", len(got), entry.size)
		}
	}
}

// A damaged cached image falls back to the source and replaces the bad file.
func TestArtDiskCacheRecoversFromCorruption(t *testing.T) {
	data := diskTestJPEG(t)
	var requests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		_, _ = w.Write(data)
	}))
	defer srv.Close()
	posts := make(chan func(), 4)
	c := newArtCache(func(fn func()) { posts <- fn })
	c.disk = newArtDiskCache(t.TempDir())
	key := sized(srv.URL+"/cover.jpg", 96)
	diskKey, _ := artDiskKey(key)
	c.disk.put(diskKey, []byte("corrupt"))
	c.get(key, false)
	runPost(t, posts)
	if a := c.get(key, false); a == nil || a.failed || requests.Load() != 1 {
		t.Fatal("the cover did not recover from a corrupt cached file")
	}
	if _, _, err := image.DecodeConfig(bytes.NewReader(c.disk.read(diskKey))); err != nil {
		t.Fatalf("the damaged cache file was not replaced: %v", err)
	}
}

// Old entries expire, cached files stay private, and the disk limit includes
// entries left by an earlier run.
func TestArtDiskCacheExpiryAndLimit(t *testing.T) {
	dir := t.TempDir()
	d := newArtDiskCache(dir)
	d.limit = 1000
	for _, key := range []string{strings.Repeat("a", 64), strings.Repeat("b", 64)} {
		d.put(key, make([]byte, 400))
	}
	first := strings.Repeat("a", 64)
	info, err := os.Stat(d.path(first))
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("cached file permissions: %v, %v", info, err)
	}
	old := time.Now().Add(-artDiskAge - time.Hour)
	if err := os.Chtimes(d.path(first), old, old); err != nil {
		t.Fatal(err)
	}
	if d.read(first) != nil {
		t.Fatal("an expired cover was reused")
	}
	next := newArtDiskCache(dir)
	next.limit = 1000
	last := strings.Repeat("c", 64)
	next.put(last, make([]byte, 400))
	if next.size > next.limit || len(next.read(last)) != 400 {
		t.Fatalf("disk cache size %d (limit %d), newest entry missing", next.size, next.limit)
	}
	if _, err := os.Stat(filepath.Join(dir, first+".img")); !os.IsNotExist(err) {
		t.Fatalf("the oldest entry was not evicted: %v", err)
	}
}

// Different credentials and requested sizes do not share cached images;
// local files bypass the disk cache so edits are seen immediately.
func TestArtDiskKeySeparatesRequests(t *testing.T) {
	previous := music.ImageHeaders
	defer func() { music.ImageHeaders = previous }()
	credential := "first secret"
	music.ImageHeaders = func(*url.URL) map[string]string { return map[string]string{"Authorization": credential} }
	first, ok := artDiskKey(sized("https://example.com/cover?token=secret", 96))
	if !ok {
		t.Fatal("network URL was not cacheable")
	}
	credential = "second secret"
	second, _ := artDiskKey(sized("https://example.com/cover?token=secret", 96))
	otherSize, _ := artDiskKey(sized("https://example.com/cover?token=secret", 600))
	if first == second || second == otherSize || strings.Contains(first, "secret") {
		t.Fatal("different image requests shared a key or exposed credentials")
	}
	if _, ok := artDiskKey(sized(music.FileURL("/tmp/cover.jpg"), 96)); ok {
		t.Fatal("a local cover was cached across file changes")
	}
}
