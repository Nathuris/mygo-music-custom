package app

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"mygo-music/internal/music"
)

const (
	artDiskLimit = 128 << 20
	artDiskItem  = 2 << 20
	artDiskAge   = 7 * 24 * time.Hour
)

// artDiskCache keeps small, already resized network covers across launches.
// Its operations run on artwork workers, never on the UI thread.
type artDiskCache struct {
	dir   string
	limit int64
	now   func() time.Time
	mu    sync.Mutex
	ready bool
	size  int64
	files map[string]artDiskFile
}

type artDiskFile struct {
	size int64
	mod  time.Time
}

func newArtDiskCache(dir string) *artDiskCache {
	return &artDiskCache{dir: dir, limit: artDiskLimit, now: time.Now}
}

// artDiskKey includes the request headers as well as the URL and size, so
// one account cannot pick up a picture fetched with another's credentials.
// Neither URLs nor headers are written in clear text to the cache directory.
func artDiskKey(key string) (string, bool) {
	u, err := url.Parse(fetchURL(key))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return "", false
	}
	hash := sha256.New()
	write := func(s string) {
		var size [8]byte
		binary.LittleEndian.PutUint64(size[:], uint64(len(s)))
		hash.Write(size[:])
		io.WriteString(hash, s)
	}
	write("mygo-cover-v1")
	write(key)
	if music.ImageHeaders != nil {
		headers := music.ImageHeaders(u)
		keys := make([]string, 0, len(headers))
		for name := range headers {
			keys = append(keys, name)
		}
		slices.Sort(keys)
		for _, name := range keys {
			write(name)
			write(headers[name])
		}
	}
	return hex.EncodeToString(hash.Sum(nil)), true
}

func (d *artDiskCache) path(key string) string { return filepath.Join(d.dir, key+".img") }

// read rejects old or oversized files; a corrupt image is retried from its
// source by the caller after decoding fails.
func (d *artDiskCache) read(key string) []byte {
	return d.readFile(key, false)
}

func (d *artDiskCache) readFile(key string, allowStale bool) []byte {
	f, err := os.Open(d.path(key))
	if err != nil {
		return nil
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > artDiskItem || !allowStale && d.now().Sub(info.ModTime()) > artDiskAge {
		return nil
	}
	b, err := io.ReadAll(io.LimitReader(f, artDiskItem+1))
	if err != nil || len(b) == 0 || len(b) > artDiskItem {
		return nil
	}
	return b
}

// index counts files once, on the first write, so the limit also covers
// pictures saved by earlier runs. The caller holds d.mu.
func (d *artDiskCache) index() {
	if d.ready {
		return
	}
	d.ready = true
	d.files = make(map[string]artDiskFile)
	entries, _ := os.ReadDir(d.dir)
	for _, entry := range entries {
		name := entry.Name()
		if len(name) != 68 || !strings.HasSuffix(name, ".img") {
			continue
		}
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() {
			continue
		}
		key := strings.TrimSuffix(name, ".img")
		d.files[key] = artDiskFile{size: info.Size(), mod: info.ModTime()}
		d.size += info.Size()
	}
}

func (d *artDiskCache) remove(key string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.index()
	if err := os.Remove(d.path(key)); err == nil || os.IsNotExist(err) {
		if old, ok := d.files[key]; ok {
			d.size -= old.size
			delete(d.files, key)
		}
	}
}

// put uses a private temporary file and atomic rename. Old entries go when
// the disk limit is reached; failing to cache never stops a cover loading.
func (d *artDiskCache) put(key string, data []byte) {
	if len(data) == 0 || len(data) > artDiskItem || int64(len(data)) > d.limit {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.index()
	if err := os.MkdirAll(d.dir, 0o700); err != nil {
		return
	}
	f, err := os.CreateTemp(d.dir, ".cover-*")
	if err != nil {
		return
	}
	defer os.Remove(f.Name())
	_, writeErr := f.Write(data)
	closeErr := f.Close()
	if writeErr != nil || closeErr != nil || os.Rename(f.Name(), d.path(key)) != nil {
		return
	}
	if old, ok := d.files[key]; ok {
		d.size -= old.size
	}
	d.files[key] = artDiskFile{size: int64(len(data)), mod: d.now()}
	d.size += int64(len(data))
	if d.size <= d.limit {
		return
	}
	keys := make([]string, 0, len(d.files))
	for k := range d.files {
		if k != key {
			keys = append(keys, k)
		}
	}
	slices.SortFunc(keys, func(a, b string) int { return d.files[a].mod.Compare(d.files[b].mod) })
	for _, k := range keys {
		if d.size <= max(d.limit*3/4, int64(len(data))) {
			break
		}
		if err := os.Remove(d.path(k)); err == nil || os.IsNotExist(err) {
			d.size -= d.files[k].size
			delete(d.files, k)
		}
	}
}
