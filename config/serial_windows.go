//go:build windows

package config

/*
extern int GetMachineGuid(char *buf, int bufLen);
*/
import "C"

import "strings"

// macSerialNumber returns a unique machine identifier on Windows.
// Uses the MachineGuid from the registry which is stable per Windows installation.
func macSerialNumber() string {
	var buf [64]C.char
	if C.GetMachineGuid(&buf[0], C.int(len(buf))) != 0 {
		return "unknown"
	}

	guid := C.GoString(&buf[0])

	// Return first 16 chars of GUID as identifier (lowercase)
	if len(guid) > 16 {
		guid = guid[:16]
	}
	return strings.ToLower(guid)
}
