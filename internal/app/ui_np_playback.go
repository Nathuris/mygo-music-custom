package app

import (
	"github.com/egoist/mygo/ui"
	"mygo-music/internal/music"
)

// nowPlayingBody keeps playback in one stable layer. Only the content panes
// are replaced and crossfaded; their transitions never lay out the controls.
func (a *app) nowPlayingBody(c *ui.Context, tr *music.Track, art *artwork, accent ui.Color, w, h float32, pointerIn bool) {
	if tr.ShowID != "" || a.npPane != paneLyrics && a.npPane != paneStage && a.npPane != paneQueue {
		a.nowPlayingBodyLegacy(c, tr, art, accent, w, h, pointerIn)
		return
	}
	a.commentsFor(tr)
	root := ui.Box(c.Key("playback-scene")).Grow(1)
	stageTarget, centerTarget := float32(0), float32(0)
	if a.npPane == paneStage {
		stageTarget = 1
	}
	if a.settings.HideLyrics && a.npPane != paneQueue {
		centerTarget = 1
	}
	stage := root.AnimateTo("stage", stageTarget, npLayout)
	center := root.AnimateTo("center", centerTarget, npLayout)
	coverSize := min(w*0.36, 440, h-436)
	side := max(coverSize, 280)
	x := lerp(npSideX(w), (w-side)/2, center)
	infoHeight := float32(48)
	if a.pl.Radio() != "" {
		infoHeight += 24
	}
	y := topBar(c) + 48 + (h-topBar(c)-60-coverSize-infoHeight-201)/2
	root.Children(func() {
		// This is the same cover element in all three panes, with no page
		// transition or shared-element transform in its ancestor chain.
		cover := ui.Box(c.Key("cover")).Absolute().Left(x+(side-coverSize)/2*center).Top(y).Size(coverSize, coverSize).
			IgnoreInput(stageTarget == 1).AccessHidden(stageTarget == 1)
		a.npMorph(c, cover).Children(func() { a.npCover(c, tr, art, coverSize, 1-stage) })
		content := ui.Column(c.Key(npPageID(a.npPane))).Absolute().Transition(npPage)
		if a.npPane == paneStage {
			content.Top(topBar(c)).Left(0).Right(0).Bottom(100).Children(func() { a.stage(c, tr, accent) })
		} else if a.npPane != paneLyrics || center < 1 {
			if a.npPane == paneLyrics {
				// Keep the old lyrics until their exit finishes, but never let
				// an invisible or departing line take a click.
				inactive := a.settings.HideLyrics || center >= 0.55
				content.Opacity(1-smoothstep(0, 0.55, center)).Translate(32*center, 0).
					IgnoreInput(inactive).Disabled(inactive).AccessHidden(inactive)
			}
			content.Left(npSideX(w) + side + 72).Right(npSideX(w)).Top(y).Bottom(44).Children(func() {
				if a.npPane == paneQueue {
					a.queuePane(c)
				} else {
					a.lyricsPane(c, tr, h)
				}
			})
		} else {
			content.Invisible()
		}
		a.npPlaybackControls(c, tr, art, accent, w, h, x, y+coverSize, side, infoHeight, stage)
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
		if a.npPane == paneStage {
			a.stageStyleButton(c)
		}
	})
	if a.npPane == paneLyrics {
		reveal(ui.Row(c.Key("lyrics-toggle-row")).Absolute().Top(topBar(c)).Right(npSideX(w)).Height(npTabsHeight), pointerIn).
			Children(func() { a.lyricsToggle(c) })
	}
}

func (a *app) npPlaybackControls(c *ui.Context, tr *music.Track, art *artwork, accent ui.Color, w, h, x, coverBottom, side, infoHeight, stage float32) {
	pl := a.pl
	compact := h - 72
	size := lerp(28, 22, stage)
	transportWidth := 80 + 6.19*size
	leftWidth := max(120, (w-96-transportWidth)/2-70)
	ui.Row(c.Key("song-info")).Absolute().Left(lerp(x, 104, stage)).Top(lerp(coverBottom+36, compact, stage)).
		Width(lerp(side, leftWidth, stage)).Height(lerp(infoHeight, 44, stage)).Gap(8).AlignItems(ui.Center).Children(func() {
		ui.Column(c).Grow(1).Shrink(1).MinWidth(0).Gap(lerp(4, 2, stage)).Children(func() {
			if pl.Radio() != "" && stage < 0.5 {
				a.radioChip(c, pl.Radio())
			}
			ui.Text(c, tr.Name).FontSize(lerp(22, 14, stage)).FontWeight(700).SingleLine()
			a.artistLinks(c, tr, white.Alpha(0.6), white, true).FontSize(lerp(15, 12, stage)).SingleLine()
		})
		a.likeButton(c.Key("like"), tr, lerp(22, 17, stage), white.Alpha(0.7))
		a.songMenu(c.Key("more"), tr, lerp(22, 17, stage), white.Alpha(0.7)).Margin(0, -8, 0, 0)
	})
	progressY := coverBottom + 36 + infoHeight + 18
	ui.Box(c.Key("progress-position")).Absolute().Left(lerp(x, 48, stage)).Top(lerp(progressY, h-96, stage)).
		Width(lerp(side, w-96, stage)).Children(func() {
		a.scrubber(c.Key("np-scrub"), &a.npScrub, white.Alpha(0.18), white.Alpha(0.9), lerp(16, 14, stage), accent.Mix(white, 0.35))
	})
	ui.Row(c.Key("times")).Absolute().Left(x).Top(progressY + 16).Width(side).Height(16).Opacity(1 - stage).Children(func() {
		ui.Text(c, clock(pl.Position())).FontSize(11).FontFeatures("tnum").TextColor(white.Alpha(0.5))
		ui.Spacer(c)
		qualityBadge(c, pl.Stream())
		ui.Spacer(c)
		ui.Text(c, "-"+clock(pl.Duration()-pl.Position())).FontSize(11).FontFeatures("tnum").TextColor(white.Alpha(0.5))
	})
	ui.Box(c.Key("transport-position")).Absolute().Left(lerp(x, (w-transportWidth)/2, stage)).
		Top(lerp(progressY+46, compact, stage)).Width(lerp(side, transportWidth, stage)).Children(func() {
		a.transport(c, white, white.Alpha(0.45), accent, size)
	})
	if stage < 1 {
		ui.Row(c.Key("volume-position")).Absolute().Left(x).Top(progressY + 115).Width(side).Justify(ui.Center).
			Opacity(1 - stage).IgnoreInput(stage > 0).AccessHidden(stage > 0).Children(func() { a.volumeSlider(c, white.Alpha(0.7), white.Alpha(0.18), 180) })
	}

	small := ui.Box(c.Key("small-cover")).Absolute().Left(48).Top(compact).Size(44, 44).Opacity(stage).IgnoreInput(true)
	small.Children(func() {
		if bmp := bitmap(art); bmp != nil {
			artImage(c, bmp, 8, artIn(small, bmp))
		}
	})
	ui.Row(c.Key("stage-time")).Absolute().Right(48).Top(compact).Height(44).AlignItems(ui.Center).Opacity(stage).Children(func() {
		ui.Text(c, clock(pl.Position())+" / "+clock(pl.Duration())).FontSize(11.5).FontFeatures("tnum").TextColor(white.Alpha(0.5))
	})
}
