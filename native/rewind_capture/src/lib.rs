use serde::Serialize;
use std::ffi::{CStr, CString};
use std::os::raw::{c_char, c_void};

mod audio_capture;
pub mod logging;
mod recorder_api;
mod replay_buffer;

use audio_capture::list_audio_devices;
use recorder_api::{list_monitors, start_replay_buffer, ReplayRecordingConfig};
use replay_buffer::ReplayBufferHandle;

#[no_mangle]
pub extern "C" fn rewind_set_log_callback(callback: logging::LogCallback) {
    logging::set_log_callback(callback);
    logging::init_logger();
    log::info!("Rust logging initialized with Go callback");
}

#[derive(Serialize)]
struct MonitorsResult {
    success: bool,
    data: Option<serde_json::Value>,
    error: Option<String>,
}

#[no_mangle]
pub extern "C" fn rewind_get_monitors() -> *mut c_char {
    log::debug!("Getting monitor list");
    let result = match list_monitors() {
        Ok(m) => {
            log::info!("Found {} monitors", m.len());
            MonitorsResult {
                success: true,
                data: serde_json::to_value(&m).ok(),
                error: None,
            }
        }
        Err(e) => {
            let error_msg = e.to_string();
            log::error!("Failed to list monitors: {}", error_msg);
            MonitorsResult {
                success: false,
                data: None,
                error: Some(error_msg),
            }
        }
    };
    let json = serde_json::to_string(&result).unwrap_or_default();
    CString::new(json).unwrap().into_raw()
}

#[derive(Serialize)]
struct AudioDevicesResult {
    success: bool,
    data: Option<serde_json::Value>,
    error: Option<String>,
}

#[no_mangle]
pub extern "C" fn rewind_list_audio_devices() -> *mut c_char {
    log::debug!("Listing audio devices");
    let result = match list_audio_devices() {
        Ok(d) => {
            log::info!("Found {} audio devices", d.len());
            AudioDevicesResult {
                success: true,
                data: serde_json::to_value(&d).ok(),
                error: None,
            }
        }
        Err(e) => {
            let error_msg = e.to_string();
            log::error!("Failed to list audio devices: {}", error_msg);
            AudioDevicesResult {
                success: false,
                data: None,
                error: Some(error_msg),
            }
        }
    };
    let json = serde_json::to_string(&result).unwrap_or_default();
    CString::new(json).unwrap().into_raw()
}

#[derive(Serialize)]
struct InitResult {
    success: bool,
    handle: usize,
    error: Option<String>,
}

#[no_mangle]
pub extern "C" fn rewind_init(monitor_index: u32, config_json: *const c_char) -> *mut c_char {
    let result = if config_json.is_null() {
        log::error!("Config JSON is null");
        InitResult {
            success: false,
            handle: 0,
            error: Some("Configuration is null".to_string()),
        }
    } else {
        let c_str = unsafe { CStr::from_ptr(config_json) };
        let json_str = match c_str.to_str() {
            Ok(s) => s,
            Err(e) => {
                log::error!("Failed to parse config string: {:?}", e);
                return CString::new(
                    serde_json::to_string(&InitResult {
                        success: false,
                        handle: 0,
                        error: Some(format!("Invalid config encoding: {}", e)),
                    })
                    .unwrap(),
                )
                .unwrap()
                .into_raw();
            }
        };

        let config: ReplayRecordingConfig = match serde_json::from_str(json_str) {
            Ok(c) => c,
            Err(e) => {
                log::error!("Failed to parse config JSON: {}", e);
                return CString::new(
                    serde_json::to_string(&InitResult {
                        success: false,
                        handle: 0,
                        error: Some(format!("Invalid configuration: {}", e)),
                    })
                    .unwrap(),
                )
                .unwrap()
                .into_raw();
            }
        };

        log::info!("Initializing replay buffer for monitor {}", monitor_index);
        match start_replay_buffer(monitor_index as usize, config) {
            Ok(handle) => {
                log::info!("Replay buffer started successfully");
                let handle_ptr = Box::into_raw(Box::new(handle)) as usize;
                InitResult {
                    success: true,
                    handle: handle_ptr,
                    error: None,
                }
            }
            Err(e) => {
                let error_msg = e.to_string();
                log::error!("Failed to start replay buffer: {}", error_msg);
                InitResult {
                    success: false,
                    handle: 0,
                    error: Some(error_msg),
                }
            }
        }
    };

    let json = serde_json::to_string(&result).unwrap_or_default();
    CString::new(json).unwrap().into_raw()
}

#[derive(Serialize)]
struct SaveResult {
    success: bool,
    error: Option<String>,
}

#[no_mangle]
pub extern "C" fn rewind_save(handle: *mut c_void, path: *const c_char) -> *mut c_char {
    let result = if handle.is_null() || path.is_null() {
        log::error!("Save called with null handle or path");
        SaveResult {
            success: false,
            error: Some("Invalid handle or path".to_string()),
        }
    } else {
        let handle = unsafe { &*(handle as *mut ReplayBufferHandle) };

        let c_path = unsafe { CStr::from_ptr(path) };
        let path_str = match c_path.to_str() {
            Ok(s) => s,
            Err(e) => {
                log::error!("Failed to parse path string: {:?}", e);
                return CString::new(
                    serde_json::to_string(&SaveResult {
                        success: false,
                        error: Some(format!("Invalid path encoding: {}", e)),
                    })
                    .unwrap(),
                )
                .unwrap()
                .into_raw();
            }
        };

        log::info!("Saving clip to: {}", path_str);
        match handle.save(path_str) {
            Ok(_) => {
                log::info!("Clip saved successfully");
                SaveResult {
                    success: true,
                    error: None,
                }
            }
            Err(e) => {
                let error_msg = e.to_string();
                log::error!("Save failed: {}", error_msg);
                SaveResult {
                    success: false,
                    error: Some(error_msg),
                }
            }
        }
    };

    let json = serde_json::to_string(&result).unwrap_or_default();
    CString::new(json).unwrap().into_raw()
}

#[no_mangle]
pub extern "C" fn rewind_stop(handle: *mut c_void) {
    if handle.is_null() {
        log::warn!("Stop called with null handle");
        return;
    }
    log::info!("Stopping replay buffer");
    let handle_box: Box<ReplayBufferHandle> =
        unsafe { Box::from_raw(handle as *mut ReplayBufferHandle) };
    handle_box.stop();
    log::info!("Replay buffer stopped");
}

#[derive(Serialize)]
struct StatusResult {
    success: bool,
    data: Option<ReplayStatus>,
    error: Option<String>,
}

#[derive(Serialize)]
struct ReplayStatus {
    duration_secs: f64,
    segment_count: usize,
    is_active: bool,
    disk_usage_bytes: u64,
    memory_usage_bytes: u64,
}

#[no_mangle]
pub extern "C" fn rewind_get_status(handle: *mut c_void) -> *mut c_char {
    let result = if handle.is_null() {
        StatusResult {
            success: false,
            data: None,
            error: Some("Invalid handle".to_string()),
        }
    } else {
        let handle = unsafe { &*(handle as *mut ReplayBufferHandle) };

        let status = ReplayStatus {
            duration_secs: handle.duration(),
            segment_count: handle.segment_count(),
            is_active: !handle
                .stop_signal()
                .load(std::sync::atomic::Ordering::Relaxed),
            disk_usage_bytes: handle.disk_usage_bytes(),
            memory_usage_bytes: handle.memory_usage_bytes(),
        };

        StatusResult {
            success: true,
            data: Some(status),
            error: None,
        }
    };

    let json = serde_json::to_string(&result).unwrap_or_default();
    CString::new(json).unwrap().into_raw()
}

#[no_mangle]
pub extern "C" fn rewind_free_string(s: *mut c_char) {
    if s.is_null() {
        return;
    }
    unsafe {
        let _ = CString::from_raw(s);
    };
}
