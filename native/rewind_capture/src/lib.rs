use serde::Serialize;
use std::ffi::{CStr, CString};
use std::os::raw::{c_char, c_void};
use std::ptr;

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

#[no_mangle]
pub extern "C" fn rewind_get_monitors() -> *mut c_char {
    log::debug!("Getting monitor list");
    let monitors = match list_monitors() {
        Ok(m) => {
            log::info!("Found {} monitors", m.len());
            m
        }
        Err(e) => {
            log::error!("Failed to list monitors: {:?}", e);
            return ptr::null_mut();
        }
    };
    let json = serde_json::to_string(&monitors).unwrap_or_default();
    CString::new(json).unwrap().into_raw()
}

#[no_mangle]
pub extern "C" fn rewind_list_audio_devices() -> *mut c_char {
    log::debug!("Listing audio devices");
    let devices = match list_audio_devices() {
        Ok(d) => {
            log::info!("Found {} audio devices", d.len());
            d
        }
        Err(e) => {
            log::error!("Failed to list audio devices: {:?}", e);
            return ptr::null_mut();
        }
    };
    let json = serde_json::to_string(&devices).unwrap_or_default();
    CString::new(json).unwrap().into_raw()
}

#[no_mangle]
pub extern "C" fn rewind_init(monitor_index: u32, config_json: *const c_char) -> *mut c_void {
    if config_json.is_null() {
        log::error!("Config JSON is null");
        return ptr::null_mut();
    }

    let c_str = unsafe { CStr::from_ptr(config_json) };
    let json_str = match c_str.to_str() {
        Ok(s) => s,
        Err(e) => {
            log::error!("Failed to parse config string: {:?}", e);
            return ptr::null_mut();
        }
    };

    let config: ReplayRecordingConfig = match serde_json::from_str(json_str) {
        Ok(c) => c,
        Err(e) => {
            log::error!("Failed to parse config JSON: {}", e);
            return ptr::null_mut();
        }
    };

    log::info!("Initializing replay buffer for monitor {}", monitor_index);
    match start_replay_buffer(monitor_index as usize, config) {
        Ok(handle) => {
            log::info!("Replay buffer started successfully");
            Box::into_raw(Box::new(handle)) as *mut c_void
        }
        Err(e) => {
            log::error!("Failed to start replay buffer: {:?}", e);
            ptr::null_mut()
        }
    }
}

#[no_mangle]
pub extern "C" fn rewind_save(handle: *mut c_void, path: *const c_char) -> i32 {
    if handle.is_null() || path.is_null() {
        log::error!("Save called with null handle or path");
        return -1;
    }

    let handle = unsafe { &*(handle as *mut ReplayBufferHandle) };

    let c_path = unsafe { CStr::from_ptr(path) };
    let path_str = match c_path.to_str() {
        Ok(s) => s,
        Err(e) => {
            log::error!("Failed to parse path string: {:?}", e);
            return -2;
        }
    };

    log::info!("Saving clip to: {}", path_str);
    match handle.save(path_str) {
        Ok(_) => {
            log::info!("Clip saved successfully");
            0
        }
        Err(e) => {
            log::error!("Save failed: {:?}", e);
            1
        }
    }
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
struct ReplayStatus {
    duration_secs: f64,
    segment_count: usize,
    is_active: bool,
    disk_usage_bytes: u64,
    memory_usage_bytes: u64,
}

#[no_mangle]
pub extern "C" fn rewind_get_status(handle: *mut c_void) -> *mut c_char {
    if handle.is_null() {
        return ptr::null_mut();
    }
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

    let json = serde_json::to_string(&status).unwrap_or_default();
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
