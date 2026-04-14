#include <CoreMediaIO/CMIOHardware.h>
#include <stdlib.h>

// IsCameraActive checks whether any CoreMediaIO device is currently running.
// Returns 1 if at least one camera device is active, 0 otherwise.
int IsCameraActive() {
    CMIOObjectPropertyAddress propertyAddress = {
        kCMIOHardwarePropertyDevices,
        kCMIOObjectPropertyScopeGlobal,
        kCMIOObjectPropertyElementMain
    };

    UInt32 dataSize = 0;
    OSStatus status = CMIOObjectGetPropertyDataSize(
        kCMIOObjectSystemObject,
        &propertyAddress,
        0,
        NULL,
        &dataSize
    );
    if (status != kCMIOHardwareNoError) {
        return 0;
    }

    UInt32 deviceCount = dataSize / sizeof(CMIODeviceID);
    if (deviceCount == 0) {
        return 0;
    }

    CMIODeviceID *devices = (CMIODeviceID *)malloc(dataSize);
    if (devices == NULL) {
        return 0;
    }

    UInt32 dataUsed = 0;
    status = CMIOObjectGetPropertyData(
        kCMIOObjectSystemObject,
        &propertyAddress,
        0,
        NULL,
        dataSize,
        &dataUsed,
        devices
    );
    if (status != kCMIOHardwareNoError) {
        free(devices);
        return 0;
    }

    // Recalculate device count from actual bytes returned.
    // Devices may have been added or removed between GetPropertyDataSize
    // and GetPropertyData (e.g. Thunderbolt dock disconnect).
    deviceCount = dataUsed / sizeof(CMIODeviceID);

    int result = 0;

    for (UInt32 i = 0; i < deviceCount; i++) {
        CMIOObjectPropertyAddress runningAddress = {
            kCMIODevicePropertyDeviceIsRunningSomewhere,
            kCMIOObjectPropertyScopeGlobal,
            kCMIOObjectPropertyElementMain
        };

        UInt32 isRunning = 0;
        UInt32 runningSize = sizeof(isRunning);
        UInt32 runningUsed = 0;
        status = CMIOObjectGetPropertyData(
            devices[i],
            &runningAddress,
            0,
            NULL,
            runningSize,
            &runningUsed,
            &isRunning
        );
        if (status == kCMIOHardwareNoError && isRunning) {
            result = 1;
            break;
        }
    }

    free(devices);
    return result;
}
