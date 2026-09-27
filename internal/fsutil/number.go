package fsutil

import (
	"crypto/rand"
	"math"
)

// Clamp clamps num between minimum and maximum.
func Clamp(num, minimum, maximum int) int {
	if num < minimum {
		return minimum
	}
	if num > maximum {
		return maximum
	}
	return num
}

var byteSizeSuffixes = []string{"B", "KB", "MB", "GB", "TB", "PB", "EB", "ZB", "YB"}

// HumanByteSize converts number (an int, int64 or float64; nil is treated as
// 0) to a human-readable byte size, rounded to precision decimal places.
func HumanByteSize(number any, precision int) (float64, string) {
	var value float64
	switch v := number.(type) {
	case float64:
		value = v
	case int:
		value = float64(v)
	case int64:
		value = float64(v)
	}

	suffix := "B"
	for _, s := range byteSizeSuffixes {
		if value < 1024 {
			suffix = s
			break
		}
		value /= 1024
	}

	scale := math.Pow(10, float64(precision))
	return math.Round(value*scale) / scale, suffix
}

// AddJitter adds a random amount, up to jitterPercentage of num, to num.
// Returns 0 if num is less than or equal to 0.
func AddJitter(num int, jitterPercentage float64) int {
	if num <= 0 {
		return 0
	}

	maxJitter := int(math.Round(float64(num) * jitterPercentage))
	if maxJitter <= 0 {
		return num
	}

	randBytes := make([]byte, 8)
	if _, err := rand.Read(randBytes); err != nil {
		return num
	}
	var randVal uint64
	for _, b := range randBytes {
		randVal = (randVal << 8) | uint64(b)
	}

	return num + int(randVal%uint64(maxJitter))
}
