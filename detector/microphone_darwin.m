#include <CoreAudio/CoreAudio.h>

// IsMicrophoneActive checks whether any audio input device is currently running.
// Returns 1 if at least one input device is active, 0 otherwise.
int IsMicrophoneActive() {
    AudioObjectPropertyAddress propertyAddress = {
        kAudioHardwarePropertyDevices,
        kAudioObjectPropertyScopeGlobal,
        kAudioObjectPropertyElementMain
    };

    UInt32 dataSize = 0;
    OSStatus status = AudioObjectGetPropertyDataSize(
        kAudioObjectSystemObject,
        &propertyAddress,
        0,
        NULL,
        &dataSize
    );
    if (status != kAudioHardwareNoError) {
        return 0;
    }

    UInt32 deviceCount = dataSize / sizeof(AudioDeviceID);
    if (deviceCount == 0) {
        return 0;
    }

    AudioDeviceID *devices = (AudioDeviceID *)malloc(dataSize);
    if (devices == NULL) {
        return 0;
    }

    status = AudioObjectGetPropertyData(
        kAudioObjectSystemObject,
        &propertyAddress,
        0,
        NULL,
        &dataSize,
        devices
    );
    if (status != kAudioHardwareNoError) {
        free(devices);
        return 0;
    }

    // Recalculate device count from actual bytes returned.
    // Devices may have been added or removed between GetPropertyDataSize
    // and GetPropertyData (e.g. Thunderbolt dock disconnect).
    deviceCount = dataSize / sizeof(AudioDeviceID);

    int result = 0;

    for (UInt32 i = 0; i < deviceCount; i++) {
        // Check if this device has input streams (i.e., is an input device).
        AudioObjectPropertyAddress streamAddress = {
            kAudioDevicePropertyStreams,
            kAudioDevicePropertyScopeInput,
            kAudioObjectPropertyElementMain
        };

        UInt32 streamSize = 0;
        status = AudioObjectGetPropertyDataSize(
            devices[i],
            &streamAddress,
            0,
            NULL,
            &streamSize
        );
        if (status != kAudioHardwareNoError || streamSize == 0) {
            continue; // Not an input device or error; skip.
        }

        // This is an input device. Check if it's running.
        AudioObjectPropertyAddress runningAddress = {
            kAudioDevicePropertyDeviceIsRunningSomewhere,
            kAudioObjectPropertyScopeGlobal,
            kAudioObjectPropertyElementMain
        };

        UInt32 isRunning = 0;
        UInt32 runningSize = sizeof(isRunning);
        status = AudioObjectGetPropertyData(
            devices[i],
            &runningAddress,
            0,
            NULL,
            &runningSize,
            &isRunning
        );
        if (status == kAudioHardwareNoError && isRunning) {
            result = 1;
            break;
        }
    }

    free(devices);
    return result;
}
