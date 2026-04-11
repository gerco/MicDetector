package detector

/*
#cgo LDFLAGS: -framework CoreMediaIO
#include <stdlib.h>

extern int IsCameraActive();
*/
import "C"

// IsCameraOn returns true if any camera device is currently in use.
func IsCameraOn() bool {
	return C.IsCameraActive() == 1
}
