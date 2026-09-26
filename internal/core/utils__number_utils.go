package core

// NumberUtilsClamp(num, minimum, maximum)
// Clamps a number between a minimum and maximum value.
func NumberUtilsClamp(num int, minimum int, maximum int) int {
	panic("unported: Pinchflat.Utils.NumberUtils.clamp/3")
}

// NumberUtilsHumanByteSize(number, opts)
// Converts a number to a human readable byte size.
// opts can include a "precision" key specifying number of decimal places.
func NumberUtilsHumanByteSize(number any, opts KW) (float64, string) {
	panic("unported: Pinchflat.Utils.NumberUtils.human_byte_size/2")
}

// NumberUtilsAddJitter(num, jitter_percentage)
// Adds jitter to a number based on a percentage.
// Returns 0 if the number is less than or equal to 0.
func NumberUtilsAddJitter(num int, jitterPercentage float64) int {
	panic("unported: Pinchflat.Utils.NumberUtils.add_jitter/2")
}
