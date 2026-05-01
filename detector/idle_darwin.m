#include <CoreGraphics/CoreGraphics.h>

// IdleSeconds returns the number of seconds since the last HID input event
// (mouse move/click, key press, etc.) seen by the system event tap.
double IdleSeconds() {
    return (double)CGEventSourceSecondsSinceLastEventType(
        kCGEventSourceStateHIDSystemState,
        kCGAnyInputEventType
    );
}
