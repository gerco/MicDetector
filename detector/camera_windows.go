//go:build windows

package detector

/*
#cgo LDFLAGS: -lole32 -loleaut32

extern int IsCameraActive(void);
*/
import "C"

// IsCameraOn returns true if any camera device is detected.
// Note: On Windows, this currently detects camera presence rather than
// actual "in-use" state. Full in-use detection requires additional work
// with device property queries or process enumeration.
func IsCameraOn() bool {
	return C.IsCameraActive() == 1
}
