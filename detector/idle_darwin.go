package detector

/*
#cgo LDFLAGS: -framework CoreGraphics
#include <stdlib.h>

extern double IdleSeconds();
*/
import "C"

// IdleSecondsNow returns the number of seconds since the last HID input event.
func IdleSecondsNow() float64 {
	return float64(C.IdleSeconds())
}
