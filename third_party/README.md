# MyGo rendering fix

`mygo/` is the source snapshot of `github.com/egoist/mygo`
`v0.8.4-0.20261107140116-94f22c384d0e`, retained locally through the root
`go.mod` replacement so normal development and release builds use the fix.
The upstream license is in `mygo/LICENSE`.

Local changes:
- `ui/effect.go`: always translate descendant transforms into an offscreen
  layer's coordinate space, including when the layer root has no transform;
  restore the original transforms after drawing.
- `ui/effect_test.go`: `TestOpacityLayerKeepsScaledChildInPlace` reproduces
  the misplaced coverage without requiring any animation or application code.

The original code skipped this conversion for an identity root. In the music
player, the cover's entry scale reaches identity before its fade ends, while
its child remains scaled to 0.94. The missing conversion offsets that child
by `(1 - childScale) * layerOrigin` until the opacity layer is removed.
The app's original left/right transitions, timing, and gesture logic are unchanged.

When an upstream version includes the fix, update the dependency, remove the
replacement and this snapshot, and run the framework regression and the app's
`TestCoverSlidePaintCoverage` against that version before removing the snapshot.
