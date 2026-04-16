//go:build windows

package detector

/*
#cgo LDFLAGS: -lole32

extern int IsMicrophoneActive(void);
*/
import "C"

// IsMicrophoneOn returns true if any audio input device is currently in use.
func IsMicrophoneOn() bool {
	return C.IsMicrophoneActive() == 1
}
