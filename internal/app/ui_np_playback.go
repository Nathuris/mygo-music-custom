package app

import (
	"time"

	"github.com/egoist/mygo/ui"
	"mygo-music/internal/lyrics"
	"mygo-music/internal/music"
)

// nowPlayingBody keeps playback in one stable layer. Only the content panes
// are replaced and crossfaded; their transitions never lay out the controls.
func (a *app) nowPlayingBody(c *ui.Context, tr *music.Track, art *artwork, accent ui.Color, w, h float32, pointerIn bool) {
	if tr.ShowID == "" && (a.npPane == paneShow || a.npPane == paneCaptions) {
		a.npPane = a.lyricsPaneChosen()
	}
	if tr.ShowID != "" {
		a.nowPlayingBodyLegacy(c, tr, art, accent, w, h, pointerIn)
		return
	}
	root := ui.Box(c.Key("playback-scene")).Grow(1)
	centerTarget := float32(0)
	if a.npPane == paneCover {
		centerTarget = 1
	}
	center := root.AnimateTo("center", centerTarget, npLayout)
	wallTarget, bottomTarget := float32(0), centerTarget
	if a.npPane == paneWall {
		wallTarget, bottomTarget = 1, 1
	}
	wallMix := root.AnimateTo("wall-layout", wallTarget, npLayout)
	bottomMix := root.AnimateTo("bottom-layout", bottomTarget, npLayout)
	fade := ui.Animation{Duration: 240 * time.Millisecond}
	lyricsTarget, queueTarget := float32(0), float32(0)
	if a.npPane == paneLyrics {
		lyricsTarget = 1
	}
	if a.npPane == paneQueue {
		queueTarget = 1
	}
	lyricsAlpha := root.AnimateTo("lyrics-content", lyricsTarget, fade)
	queueAlpha := root.AnimateTo("queue-content", queueTarget, fade)
	wallAlpha := root.AnimateTo("wall-content", wallTarget, fade)
	normalCover := min(w*0.36, 440, h-436)
	compactCover := min(w*0.32, 440, h*0.49)
	wallTop := topBar(c) + npTabsHeight
	wallSongHeight := min(max(h*0.36, 200), 320)
	wallCover := wallSongHeight - 48
	wallTextWidth := max(0, min(440, w-144-wallCover-36))
	wallX := (w - wallCover - 36 - wallTextWidth) / 2
	wallY := wallTop + 24
	coverSize := normalCover*(1-center-wallMix) + compactCover*center + wallCover*wallMix
	side := max(normalCover, 280)
	x := lerp(npSideX(w), (w-side)/2, center)
	infoHeight := float32(48)
	if a.pl.Radio() != "" {
		infoHeight += 24
	}
	normalY := topBar(c) + 48 + (h-topBar(c)-60-normalCover-infoHeight-201)/2
	compactY := max(topBar(c)+50, (h-compactCover)*0.36)
	y := normalY*(1-center-wallMix) + compactY*center + wallY*wallMix
	root.Children(func() {
		if wallAlpha > 0 {
			ui.Box(c.Key("wall-content")).Absolute().Top(0).Left(0).Right(0).Bottom(0).
				Opacity(wallAlpha).IgnoreInput(a.npPane != paneWall).AccessHidden(a.npPane != paneWall).
				Children(func() { a.wallBackground(c, w, h) })
		} else if a.npPane != paneWall {
			a.wallState.track = ""
		}
		if lyricsAlpha > 0 {
			inactive := a.npPane != paneLyrics
			ui.Column(c.Key("lyrics-content")).Absolute().Left(npSideX(w)+side+72).Right(npSideX(w)).Top(normalY).Bottom(44).
				Opacity(lyricsAlpha*(1-smoothstep(0, 0.55, center))).Translate(32*center, 0).
				IgnoreInput(inactive).Disabled(inactive).AccessHidden(inactive).
				Children(func() { a.lyricsPane(c, tr, h) })
		}
		if queueAlpha > 0 {
			inactive := a.npPane != paneQueue
			ui.Column(c.Key("queue-content")).Absolute().Left(npSideX(w) + side + 72).Right(npSideX(w)).Top(normalY).Bottom(44).
				Opacity(queueAlpha).IgnoreInput(inactive).Disabled(inactive).AccessHidden(inactive).
				Children(func() { a.queuePane(c) })
		} else if a.npPane != paneQueue {
			a.queueAt = -1
		}
		// Keep the moving cover above fading content, without a page
		// transition or a group opacity in its ancestor chain.
		coverX := npSideX(w)*(1-center-wallMix) + (w-compactCover)/2*center + wallX*wallMix
		cover := ui.Box(c.Key("cover")).Absolute().Left(coverX).Top(y).Size(coverSize, coverSize).IgnoreInput(a.npPane == paneWall)
		a.npMorph(c, cover).Children(func() { a.npCover(c, tr, art, coverSize) })
		if center > 0 {
			a.compactLyrics(c, tr, y+coverSize, center)
		}
		if wallAlpha > 0 {
			infoX, infoY := npInfoPosition(x, normalY+normalCover+36, h-72, bottomMix, wallMix, wallX+wallCover+36, wallTop+(wallSongHeight-100)/2)
			a.wallLyrics(c, infoX, infoY+54, wallTextWidth, wallAlpha*smoothstep(0.7, 1, wallMix))
		}
		a.npPlaybackControls(c, tr, art, accent, w, h, x, normalY+normalCover, side, infoHeight, bottomMix, center, wallMix, wallX+wallCover+36, wallTop+(wallSongHeight-100)/2, wallTextWidth)
	})
	a.npRise(ui.Row(c.Key("np-close-strip")).Absolute().Top(0).Left(0).Right(0).Height(topBar(c)).Justify(ui.Center).AlignItems(ui.Center).DragWindow(), 0).Children(func() {
		reveal(ui.Box(c.Key("close-reveal")), pointerIn).Children(func() {
			if iconButton(c.Key("close"), iconDown, 22, white.Alpha(0.7), T("Close")).Clicked() {
				a.nowPlaying = false
			}
		})
	})
	a.npRise(reveal(npTabsRow(c, w), pointerIn), 0).Children(func() {
		a.paneTabs(c)
	})
}

// compactLyrics shows the song's credits and current line under its cover.
func (a *app) compactLyrics(c *ui.Context, tr *music.Track, coverBottom, shown float32) {
	lines, loaded := a.pl.Lyrics()
	pos := a.pl.Position()
	cur := lyrics.Index(lines, pos)
	w, h := c.Size()
	scale := min(1, max(0.55, (h-coverBottom-120)/108))
	if a.pl.Playing() && cur+1 < len(lines) {
		c.After(lines[cur+1].At - pos + 20*time.Millisecond)
	}
	creditsTop := coverBottom + 14
	creditsHeight := (32*1.15 + 2 + 14*1.15) * scale
	alpha := smoothstep(0.5, 1, shown)
	ui.Column(c.Key("compact-credits")).Absolute().Top(creditsTop).Left(48).Right(48).
		AlignItems(ui.Center).Gap(2 * scale).Opacity(alpha).IgnoreInput(true).Children(func() {
		ui.Text(c, tr.Name).FontSize(32 * scale).FontWeight(400).LineHeight(1.15).
			TextColor(white.Alpha(0.95)).TextAlign(ui.Center).MaxWidth(w - 96).SingleLine()
		if artists := tr.ArtistNames(); artists != "" {
			ui.Text(c, artists).FontSize(14 * scale).FontWeight(400).LineHeight(1.15).
				TextColor(white.Alpha(0.6)).TextAlign(ui.Center).MaxWidth(w - 96).SingleLine()
		}
	})
	ui.Column(c.Key("compact-current-line")).Absolute().Top(creditsTop + creditsHeight + 8*scale).Left(48).Right(48).
		Bottom(108).Center().Gap(4 * scale).Clip().Opacity(alpha).IgnoreInput(true).Children(func() {
		if len(lines) == 0 && loaded {
			ui.Text(c, T("No lyrics. Enjoy the music.")).FontSize(18 * scale).FontWeight(400).LineHeight(1.15).
				TextColor(white.Alpha(0.55)).TextAlign(ui.Center)
		} else if cur >= 0 {
			ui.Text(c, lines[cur].Text).FontSize(18 * scale).FontWeight(400).LineHeight(1.15).
				TextColor(white.Alpha(0.8)).TextAlign(ui.Center).MaxWidth(w - 96)
			if lines[cur].Translation != "" {
				ui.Text(c, lines[cur].Translation).FontSize(14 * scale).FontWeight(400).LineHeight(1.15).
					TextColor(white.Alpha(0.5)).TextAlign(ui.Center).MaxWidth(w - 96)
			}
		}
	})
}

// wallLyrics shows the current and next lines beside the wall's cover.
func (a *app) wallLyrics(c *ui.Context, x, y, width, shown float32) {
	lines, _ := a.pl.Lyrics()
	if len(lines) == 0 {
		return
	}
	pos := a.pl.Position()
	cur := lyrics.Index(lines, pos)
	if a.pl.Playing() && cur+1 < len(lines) {
		c.After(lines[cur+1].At - pos + 20*time.Millisecond)
	}
	ui.Column(c.Key("wall-lyrics")).Absolute().Left(x).Top(y).Width(width).Gap(4).
		Opacity(shown).IgnoreInput(true).Children(func() {
		if cur >= 0 {
			ui.Text(c, lines[cur].Text).FontSize(17).FontWeight(700).TextColor(white.Alpha(0.85)).MaxLines(2)
		}
		if cur+1 < len(lines) {
			ui.Text(c, lines[cur+1].Text).FontSize(14).FontWeight(600).TextColor(white.Alpha(0.38)).SingleLine()
		}
	})
}

// The song information moves alongside the cover before rising beside it.
func npInfoPosition(x, y, bottom, compactMix, wallMix, wallX, wallY float32) (float32, float32) {
	return lerp(lerp(x, 104, compactMix), wallX, smoothstep(0, 0.7, wallMix)),
		lerp(lerp(y, bottom, compactMix), wallY, smoothstep(0.25, 1, wallMix))
}

func (a *app) npPlaybackControls(c *ui.Context, tr *music.Track, art *artwork, accent ui.Color, w, h, x, coverBottom, side, infoHeight, compactMix, coverMix, wallMix, wallInfoX, wallInfoY, wallInfoWidth float32) {
	pl := a.pl
	bottom := h - 72
	size := lerp(28, 22, compactMix)
	transportWidth := 80 + 6.19*size
	leftWidth := max(120, (w-96-transportWidth)/2-70)
	infoX, infoY := npInfoPosition(x, coverBottom+36, bottom, compactMix, wallMix, wallInfoX, wallInfoY)
	infoWidth := lerp(lerp(side, leftWidth, compactMix), wallInfoWidth, wallMix)
	ui.Row(c.Key("song-info")).Absolute().Left(infoX).Top(infoY).
		Width(infoWidth).Height(lerp(infoHeight, 44, compactMix)).Gap(8).AlignItems(ui.Center).Children(func() {
		ui.Column(c).Grow(1).Shrink(1).MinWidth(0).Gap(lerp(4, 2, compactMix)).Children(func() {
			if pl.Radio() != "" && compactMix < 0.5 {
				a.radioChip(c, pl.Radio())
			}
			ui.Text(c, tr.Name).FontSize(lerp(lerp(22, 14, compactMix), 26, wallMix)).FontWeight(700).SingleLine()
			a.artistLinks(c, tr, white.Alpha(0.6), white, true).FontSize(lerp(15, 12, compactMix)).SingleLine()
		})
		a.likeButton(c.Key("like"), tr, lerp(lerp(22, 17, compactMix), 20, wallMix), white.Alpha(0.7))
		a.songMenu(c.Key("more"), tr, lerp(lerp(22, 17, compactMix), 20, wallMix), white.Alpha(0.7)).Margin(0, -8, 0, 0)
	})
	progressY := coverBottom + 36 + infoHeight + 18
	ui.Box(c.Key("progress-position")).Absolute().Left(lerp(x, 48, compactMix)).Top(lerp(lerp(progressY, h-96, compactMix), h-92, wallMix)).
		Width(lerp(side, w-96, compactMix)).Children(func() {
		a.scrubber(c.Key("np-scrub"), &a.npScrub, white.Alpha(0.18), white.Alpha(0.9), lerp(16, 14, compactMix), accent.Mix(white, 0.35))
	})
	ui.Row(c.Key("times")).Absolute().Left(x).Top(progressY + 16).Width(side).Height(16).Opacity(1 - compactMix).Children(func() {
		ui.Text(c, clock(pl.Position())).FontSize(11).FontFeatures("tnum").TextColor(white.Alpha(0.5))
		ui.Spacer(c)
		qualityBadge(c, pl.Stream())
		ui.Spacer(c)
		ui.Text(c, "-"+clock(pl.Duration()-pl.Position())).FontSize(11).FontFeatures("tnum").TextColor(white.Alpha(0.5))
	})
	ui.Box(c.Key("transport-position")).Absolute().Left(lerp(x, (w-transportWidth)/2, compactMix)).
		Top(lerp(lerp(progressY+46, bottom, compactMix), h-68, wallMix)).Width(lerp(side, transportWidth, compactMix)).Children(func() {
		a.transport(c, white, white.Alpha(0.35), white, size)
	})
	timeText := clock(pl.Position()) + " / " + clock(pl.Duration())
	timeWidth, _ := c.MeasureText(0, ui.Span{Text: timeText, Size: 11.5})
	volumeLeft := (w+transportWidth)/2 + 12
	volumeSpace := w - 48 - max(80, timeWidth+12) - volumeLeft
	coverVolumeWidth := min(130, max(32, volumeSpace-104))
	volumeWidth := lerp(180, coverVolumeWidth, compactMix)
	volumeRowWidth := lerp(side, coverVolumeWidth+104, compactMix)
	ui.Row(c.Key("volume-position")).Absolute().Left(lerp(x, volumeLeft, compactMix)).
		Top(lerp(progressY+115, bottom+6, compactMix)).Width(volumeRowWidth).Justify(ui.Center).
		Children(func() { a.volumeSlider(c, white.Alpha(0.7), white.Alpha(0.18), volumeWidth) })

	small := ui.Box(c.Key("small-cover")).Absolute().Left(48).Top(bottom).Size(44, 44).Opacity(coverMix).IgnoreInput(true)
	small.Children(func() {
		if bmp := bitmap(art); bmp != nil {
			artImage(c, bmp, 8, artIn(small, bmp))
		}
	})
	ui.Row(c.Key("compact-time")).Absolute().Right(48).Top(bottom).Height(44).AlignItems(ui.Center).Opacity(coverMix).Children(func() {
		ui.Text(c, timeText).FontSize(11.5).FontFeatures("tnum").TextColor(white.Alpha(0.5))
	})
	ui.Row(c.Key("wall-time")).Absolute().Left(48).Right(48).Top(h - 68).Height(44).AlignItems(ui.Center).Opacity(wallMix).IgnoreInput(true).Children(func() {
		ui.Text(c, clock(pl.Position())).FontSize(11.5).FontFeatures("tnum").TextColor(white.Alpha(0.5))
		ui.Spacer(c)
		ui.Text(c, "-"+clock(pl.Duration()-pl.Position())).FontSize(11.5).FontFeatures("tnum").TextColor(white.Alpha(0.5))
	})
}
