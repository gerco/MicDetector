//go:build darwin

package config

/*
#cgo LDFLAGS: -framework IOKit -framework CoreFoundation
#include <stdlib.h>

extern int GetSerialNumber(char *buf, int bufLen);
*/
import "C"

import "strings"

// macSerialNumber returns the Mac's serial number in lower case via IOKit.
func macSerialNumber() string {
	var buf [64]C.char
	if C.GetSerialNumber(&buf[0], C.int(len(buf))) != 0 {
		return "unknown"
	}
	return strings.ToLower(C.GoString(&buf[0]))
}
