package core

import (
	"crypto/rand"
	"math"
)

// NumberUtilsClamp(num, minimum, maximum)
// Clamps a number between a minimum and maximum value.
func NumberUtilsClamp(num int, minimum int, maximum int) int {
	if num < minimum {
		return minimum
	}
	if num > maximum {
		return maximum
	}
	return num
}

// NumberUtilsHumanByteSize(number, opts)
// Converts a number to a human readable byte size.
// opts can include a "precision" key specifying number of decimal places.
func NumberUtilsHumanByteSize(number any, opts KW) (float64, string) {
	precision := int64(2)
	if p, ok := opts.Get("precision"); ok {
		if pInt, ok := p.(int); ok {
			precision = int64(pInt)
		} else if pInt, ok := p.(int64); ok {
			precision = pInt
		}
	}

	suffixes := []string{"B", "KB", "MB", "GB", "TB", "PB", "EB", "ZB", "YB"}
	base := 1024.0

	// Convert number to float, handle nil
	var value float64
	if number == nil {
		value = 0
	} else if f, ok := number.(float64); ok {
		value = f
	} else if i, ok := number.(int); ok {
		value = float64(i)
	} else if i, ok := number.(int64); ok {
		value = float64(i)
	} else {
		value = 0
	}

	value /= 1.0
	suffix := "B"

	for _, s := range suffixes {
		if value < base {
			suffix = s
			break
		}
		value /= base
	}

	// Round to precision
	roundedValue := math.Round(value*math.Pow(10, float64(precision))) / math.Pow(10, float64(precision))

	return roundedValue, suffix
}

// NumberUtilsAddJitter(num, jitter_percentage)
// Adds jitter to a number based on a percentage.
// Returns 0 if the number is less than or equal to 0.
func NumberUtilsAddJitter(num int, jitterPercentage float64) int {
	if num <= 0 {
		return 0
	}

	maxJitter := int(math.Round(float64(num) * jitterPercentage))
	if maxJitter <= 0 {
		return num
	}

	// Generate random int between 0 and maxJitter
	randBytes := make([]byte, 8)
	_, err := rand.Read(randBytes)
	if err != nil {
		return num
	}

	// Convert bytes to uint64 and mod by maxJitter
	var randVal uint64
	for i, b := range randBytes {
		randVal = (randVal << 8) | uint64(b)
		if i >= 7 {
			break
		}
	}
	jitter := int(randVal % uint64(maxJitter))

	return num + jitter
}
