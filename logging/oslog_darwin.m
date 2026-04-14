#include <os/log.h>

static os_log_t _log = NULL;

void MicDetectorLogInit(const char *subsystem, const char *category) {
    _log = os_log_create(subsystem, category);
}

void MicDetectorLogDebug(const char *msg) {
    if (_log) os_log_with_type(_log, OS_LOG_TYPE_DEBUG, "%{public}s", msg);
}

void MicDetectorLogDefault(const char *msg) {
    if (_log) os_log_with_type(_log, OS_LOG_TYPE_DEFAULT, "%{public}s", msg);
}

void MicDetectorLogError(const char *msg) {
    if (_log) os_log_with_type(_log, OS_LOG_TYPE_ERROR, "%{public}s", msg);
}
