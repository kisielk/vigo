package editor

const (
	ConfigWrapLeft  = true // Allow wrapping cursor to the previous line with 'h' motion.
	ConfigWrapRight = true // Allow wrapping cursor to the next line with 'l' motion.

	// TODO keep synched with utils/utils.go/TabstopLength
	TabstopLength           = 8
	ViewVerticalThreshold   = 5
	ViewHorizontalThreshold = 10
)
