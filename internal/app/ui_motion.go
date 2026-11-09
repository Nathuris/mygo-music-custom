package app

import "math"

// lineRandom gives a repeatable random value for playback effects.
func lineRandom(seed, offset float64) float32 {
	x := math.Sin(seed+offset) * 10000
	return float32(x - math.Floor(x))
}

func clamp01(v float32) float32 { return min(1, max(0, v)) }

func lerp(a, b, t float32) float32 { return a + (b-a)*t }
