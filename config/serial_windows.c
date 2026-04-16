#include <windows.h>
#include <stdio.h>

// Get Windows MachineGuid from registry
// Returns 0 on success, -1 on failure
int GetMachineGuid(char *buf, int bufLen) {
    HKEY hKey;
    DWORD dwType = REG_SZ;
    DWORD dwSize = bufLen;
    LONG lResult;
    
    lResult = RegOpenKeyExA(
        HKEY_LOCAL_MACHINE,
        "SOFTWARE\\Microsoft\\Cryptography",
        0,
        KEY_QUERY_VALUE,
        &hKey
    );
    
    if (lResult != ERROR_SUCCESS) {
        return -1;
    }
    
    lResult = RegQueryValueExA(
        hKey,
        "MachineGuid",
        NULL,
        &dwType,
        (LPBYTE)buf,
        &dwSize
    );
    
    RegCloseKey(hKey);
    
    if (lResult != ERROR_SUCCESS) {
        return -1;
    }
    
    return 0;
}
