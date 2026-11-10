# Cover jitter: opacity-layer coordinate conversion

## Finding

The defect is in MyGo's `ui/effect.go: layerPaint`, not in the player queue,
artwork decoding, or the cover animation's spring. The original app animation
is retained unchanged.

An offscreen opacity group shifts its subtree to the bitmap's origin. The
original implementation converted descendant transforms only when the group
root itself had a transform (`world.Set`). An identity root can still contain
a scaled child. In that case the child's transform continued to act around
its window-space origin, rather than the offscreen bitmap's origin.

In the cover transition, its entry scale reaches 1 before its opacity reaches
1: the scale channel's smaller travel meets the spring's absolute precision
threshold earlier. Its internal breathing card remains scaled to 0.94. For
that interval the root has no transform but still needs an opacity layer.
The missed conversion shifts the image by approximately
`(1 - 0.94) * layerOrigin`. When opacity reaches 1, rendering becomes direct
again and the image returns to its correct place.

This explains the sudden displacement and return, the larger displacement on
centered pages, and why dragging did not trigger the same fault: the drag's
translation keeps the group root transformed during its fade.

## Decisive evidence

- A static framework reproducer needs no song change, spring, exit ghost,
  asynchronous work, or wall-clock phase. A 0.94-scaled image with a shadow
  changes from opacity 1 to 0.999 under an identity group root.
- Original framework: its painted coverage moves from x=306..493 to
  x=320..507. The same test passes after correcting the coordinate conversion.
- Original app regression: the incoming cover is displaced 20.28 px on Cover
  and 10.36 px on Wall (the Chinese song-picker tab). Element bounds remain
  correct, so the older layout-only tests miss this defect.
- The correction removes these displacements without changing
  `internal/app/ui_cover.go`, the entry/exit motions, spring constants,
  immediate button/keyboard playback, or the drag state machine.

## Correction and delivery

Always convert descendant transforms into the layer's coordinate space,
including for an identity root, and restore them after drawing. The pinned
MyGo source is retained in `third_party/mygo` with a relative Go module
replacement; development and normal release builds use the same correction.
See `third_party/README.md` for the upstream baseline and removal condition.

## Verification

- `TestOpacityLayerKeepsScaledChildInPlace`: fails on original code;
  passes five consecutive runs on the correction.
- `TestCoverSlidePaintCoverage`: fails on the original dependency; passes on
  the correction across Lyrics, Cover, Up Next and Wall, sampling actual
  rendered coverage through the end of the transition without forced rebuilds.
- App coverage, original button slides and original gestures: five runs pass.
- Full app tests, full framework UI tests, and MyGo UI vet pass.
- Windows and Linux application cross-builds pass; the macOS development app
  rebuilt successfully and its button and Command-Left shortcuts were exercised.
- Rendered before/after frame strips were inspected around the failing interval.
  Native interaction screenshots establish the working entry points; the
  deterministic frame-by-frame coverage checks establish removal of the transient
  rendering defect.
