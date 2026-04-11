#include <IOKit/IOKitLib.h>
#include <CoreFoundation/CoreFoundation.h>
#include <string.h>

// GetSerialNumber writes the Mac's serial number into buf (up to bufLen-1 chars).
// Returns 0 on success, -1 on failure.
int GetSerialNumber(char *buf, int bufLen) {
    io_service_t platformExpert = IOServiceGetMatchingService(
        kIOMainPortDefault,
        IOServiceMatching("IOPlatformExpertDevice")
    );
    if (platformExpert == 0) {
        return -1;
    }

    CFStringRef serialRef = IORegistryEntryCreateCFProperty(
        platformExpert,
        CFSTR(kIOPlatformSerialNumberKey),
        kCFAllocatorDefault,
        0
    );
    IOObjectRelease(platformExpert);

    if (serialRef == NULL) {
        return -1;
    }

    Boolean ok = CFStringGetCString(serialRef, buf, bufLen, kCFStringEncodingUTF8);
    CFRelease(serialRef);

    return ok ? 0 : -1;
}
