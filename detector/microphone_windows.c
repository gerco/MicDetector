#include <windows.h>
#include <mmdeviceapi.h>
#include <endpointvolume.h>
#include <stdio.h>

// Define missing GUIDs for MinGW
const IID IID_IAudioMeterInformation = {0xC02216F6, 0x8C67, 0x4B5B, {0x9D, 0x00, 0xD0, 0x08, 0xE7, 0x3E, 0x01, 0xE0}};
const CLSID CLSID_MMDeviceEnumerator = {0xBCDE0395, 0xE52F, 0x467C, {0x8E, 0x3D, 0xC4, 0x57, 0x92, 0x91, 0x69, 0x2E}};
const IID IID_IMMDeviceEnumerator = {0xA95664D2, 0x9614, 0x4F35, {0xA5, 0x46, 0xDE, 0x8D, 0xB8, 0x8E, 0x17, 0x09}};

// IAudioMeterInformation is incomplete in MinGW headers.
// We use a wrapper type and helper functions to provide a clean interface.
typedef void IAudioMeterInformation_Wrapped;

// Helper: Get peak value from audio meter (vtable index 3)
static HRESULT IAudioMeterInformation_GetPeakValue(IAudioMeterInformation_Wrapped *meter, float *peak) {
    typedef HRESULT (__stdcall *GetPeakValue_t)(void*, float*);
    return ((GetPeakValue_t)((void**)meter)[3])(meter, peak);
}

// Helper: Release audio meter (vtable index 2)
static void IAudioMeterInformation_Release(IAudioMeterInformation_Wrapped *meter) {
    ((void (__stdcall *)(void*))((void**)meter)[2])(meter);
}

// Initialize COM and check if any audio input device is active
// Returns 1 if active, 0 if not or error
int IsMicrophoneActive(void) {
    HRESULT hr;
    IMMDeviceEnumerator *pEnumerator = NULL;
    IMMDeviceCollection *pCollection = NULL;
    IMMDevice *pDevice = NULL;
    IAudioMeterInformation_Wrapped *pMeter = NULL;
    UINT count = 0;
    UINT i;
    int result = 0;
    
    // Initialize COM
    hr = CoInitializeEx(NULL, COINIT_APARTMENTTHREADED);
    if (FAILED(hr)) {
        return 0;
    }
    
    // Create device enumerator
    hr = CoCreateInstance(
        &CLSID_MMDeviceEnumerator,
        NULL,
        CLSCTX_ALL,
        &IID_IMMDeviceEnumerator,
        (void**)&pEnumerator
    );
    
    if (FAILED(hr)) {
        CoUninitialize();
        return 0;
    }
    
    // Get all audio capture (input) devices
    hr = pEnumerator->lpVtbl->EnumAudioEndpoints(pEnumerator, eCapture, DEVICE_STATE_ACTIVE, &pCollection);
    if (FAILED(hr)) {
        pEnumerator->lpVtbl->Release(pEnumerator);
        CoUninitialize();
        return 0;
    }
    
    // Get device count
    hr = pCollection->lpVtbl->GetCount(pCollection, &count);
    if (FAILED(hr) || count == 0) {
        pCollection->lpVtbl->Release(pCollection);
        pEnumerator->lpVtbl->Release(pEnumerator);
        CoUninitialize();
        return 0;
    }
    
    // Check each device to see if it's in use
    for (i = 0; i < count && result == 0; i++) {
        hr = pCollection->lpVtbl->Item(pCollection, i, &pDevice);
        if (FAILED(hr)) continue;
        
        // Try to get audio meter information
        hr = pDevice->lpVtbl->Activate(pDevice, &IID_IAudioMeterInformation, CLSCTX_ALL, NULL, (void**)&pMeter);
        
        if (SUCCEEDED(hr) && pMeter) {
            float peak = 0.0f;
            hr = IAudioMeterInformation_GetPeakValue(pMeter, &peak);
            
            // If peak value > 0, device is actively capturing
            if (SUCCEEDED(hr) && peak > 0.0f) {
                result = 1;
            }
            
            IAudioMeterInformation_Release(pMeter);
            pMeter = NULL;
        }
        
        pDevice->lpVtbl->Release(pDevice);
        pDevice = NULL;
    }
    
    pCollection->lpVtbl->Release(pCollection);
    pEnumerator->lpVtbl->Release(pEnumerator);
    CoUninitialize();
    
    return result;
}
