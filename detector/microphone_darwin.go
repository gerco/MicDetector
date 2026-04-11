package detector

/*
#cgo LDFLAGS: -framework CoreAudio
#include <stdlib.h>

extern int IsMicrophoneActive();
*/
import "C"

// IsMicrophoneOn returns true if any audio input device is currently in use.
func IsMicrophoneOn() bool {
	return C.IsMicrophoneActive() == 1
}
