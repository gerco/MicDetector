package detector

/*
#cgo LDFLAGS: -framework CoreGraphics -framework CoreFoundation
#include <stdlib.h>

extern int IsScreenLocked();
*/
import "C"

// IsScreenLockedNow returns true if the login session reports the screen as locked.
func IsScreenLockedNow() bool {
	return C.IsScreenLocked() == 1
}
