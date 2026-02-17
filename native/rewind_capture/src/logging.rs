use log::{Level, LevelFilter, Metadata, Record};
use std::ffi::CString;
use std::os::raw::c_char;
use std::sync::Mutex;

pub type LogCallback = extern "C" fn(level: i32, message: *const c_char);

static LOG_CALLBACK: Mutex<Option<LogCallback>> = Mutex::new(None);

pub struct GoLogger;

impl log::Log for GoLogger {
    fn enabled(&self, metadata: &Metadata) -> bool {
        metadata.level() <= Level::Debug
    }

    fn log(&self, record: &Record) {
        if self.enabled(record.metadata()) {
            let callback = LOG_CALLBACK.lock().unwrap();
            if let Some(cb) = *callback {
                let level = match record.level() {
                    Level::Error => 0,
                    Level::Warn => 1,
                    Level::Info => 2,
                    Level::Debug => 3,
                    Level::Trace => 4,
                };

                let msg = format!("{}", record.args());

                if let Ok(c_msg) = CString::new(msg) {
                    cb(level, c_msg.as_ptr());
                }
            }
        }
    }

    fn flush(&self) {}
}

pub fn set_log_callback(callback: LogCallback) {
    let mut cb = LOG_CALLBACK.lock().unwrap();
    *cb = Some(callback);
}

pub fn init_logger() {
    log::set_logger(&GoLogger)
        .map(|()| log::set_max_level(LevelFilter::Debug))
        .ok();
}
