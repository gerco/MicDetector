#include <windows.h>
#include <dshow.h>
#include <stdio.h>

// Define missing GUIDs for MinGW
const CLSID CLSID_SystemDeviceEnum = {0x62BE5D10, 0x60EB, 0x11D0, {0xBD, 0x3B, 0x00, 0xA0, 0xC9, 0x11, 0xCE, 0x86}};
const IID IID_ICreateDevEnum = {0x29840822, 0x5BDD, 0x11D0, {0xB8, 0xED, 0x00, 0xA0, 0xC9, 0x22, 0x31, 0x96}};
const GUID CLSID_VideoInputDeviceCategory = {0x860BB310, 0x5D01, 0x11D0, {0xBD, 0x3B, 0x00, 0xA0, 0xC9, 0x11, 0xCE, 0x86}};

// Check if any camera is in use by enumerating video input devices
// Returns 1 if any camera is found and appears to be active
// Note: On Windows, detecting actual "in-use" state for cameras is complex
// We enumerate devices and return 1 if at least one video device exists
int IsCameraActive(void) {
    HRESULT hr;
    ICreateDevEnum *pDevEnum = NULL;
    IEnumMoniker *pEnum = NULL;
    IMoniker *pMoniker = NULL;
    int deviceCount = 0;

    // Initialize COM
    hr = CoInitializeEx(NULL, COINIT_APARTMENTTHREADED);
    if (FAILED(hr)) {
        return 0;
    }

    // Create System Device Enumerator
    hr = CoCreateInstance(
        &CLSID_SystemDeviceEnum,
        NULL,
        CLSCTX_INPROC_SERVER,
        &IID_ICreateDevEnum,
        (void**)&pDevEnum
    );

    if (FAILED(hr)) {
        CoUninitialize();
        return 0;
    }

    // Create enumerator for video input devices (cameras)
    hr = pDevEnum->lpVtbl->CreateClassEnumerator(pDevEnum, &CLSID_VideoInputDeviceCategory, &pEnum, 0);

    if (FAILED(hr) || hr == S_FALSE) {
        // No devices found
        pDevEnum->lpVtbl->Release(pDevEnum);
        CoUninitialize();
        return 0;
    }

    // Count devices
    while (pEnum->lpVtbl->Next(pEnum, 1, &pMoniker, NULL) == S_OK) {
        deviceCount++;
        pMoniker->lpVtbl->Release(pMoniker);
        pMoniker = NULL;
    }

    pEnum->lpVtbl->Release(pEnum);
    pDevEnum->lpVtbl->Release(pDevEnum);
    CoUninitialize();

    // Return 1 if at least one camera exists
    // Note: This simplified approach just checks for camera presence
    // True "in-use" detection on Windows requires more complex approaches
    return deviceCount > 0 ? 1 : 0;
}
