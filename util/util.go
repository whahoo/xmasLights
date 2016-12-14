package util

import "math/rand"

func RandomFloat(min, max float64) float64 {
	return rand.Float64()*(max-min) + min
}

func Random(min, max int) uint8 {
	xr := rand.Intn(max-min) + min
	return uint8(xr)
}
