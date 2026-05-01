#include <CoreFoundation/CoreFoundation.h>

// CGSessionCopyCurrentDictionary is exported by CoreGraphics but not declared
// in any public header, so we forward-declare it here.
extern CFDictionaryRef CGSessionCopyCurrentDictionary(void);

// IsScreenLocked returns 1 if the login session reports the screen as locked,
// 0 otherwise (including when the dictionary cannot be read).
int IsScreenLocked() {
    CFDictionaryRef session = CGSessionCopyCurrentDictionary();
    if (session == NULL) {
        return 0;
    }

    int result = 0;
    CFTypeRef value = CFDictionaryGetValue(session, CFSTR("CGSSessionScreenIsLocked"));
    if (value != NULL && CFGetTypeID(value) == CFBooleanGetTypeID()) {
        if (CFBooleanGetValue((CFBooleanRef)value)) {
            result = 1;
        }
    }

    CFRelease(session);
    return result;
}
