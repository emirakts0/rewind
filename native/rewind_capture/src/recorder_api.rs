use serde::{Deserialize, Serialize};
use std::sync::atomic::{self, AtomicBool};
use std::sync::{Arc, Mutex};
use std::time::{Duration, Instant};

use windows_capture::capture::{Context, GraphicsCaptureApiHandler};
use windows_capture::encoder::{
    AudioSettingsBuilder, ContainerSettingsBuilder, ContainerSettingsSubType, VideoEncoder,
    VideoSettingsBuilder, VideoSettingsSubType,
};
use windows_capture::frame::Frame;
use windows_capture::graphics_capture_api::InternalCaptureControl;
use windows_capture::monitor::Monitor;
use windows_capture::settings::{
    ColorFormat, CursorCaptureSettings, DirtyRegionSettings, DrawBorderSettings,
    MinimumUpdateIntervalSettings, SecondaryWindowSettings, Settings,
};

use crate::audio_capture::{AudioCapture, AudioCaptureConfig, CircularAudioBuffer};
use crate::replay_buffer::{ReplayBuffer, ReplayBufferConfig, ReplayBufferHandle};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MonitorInfo {
    pub index: usize,
    pub name: String,
    pub width: u32,
    pub height: u32,
    pub refresh_rate: u32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AudioConfig {
    pub sample_rate: u32,
    pub channels: u16,
    pub mic_enabled: bool,
    pub mic_device_index: Option<usize>,
    pub mic_volume: f32,
    pub speaker_enabled: bool,
    pub speaker_device_index: Option<usize>,
    pub speaker_volume: f32,
}

impl Default for AudioConfig {
    fn default() -> Self {
        Self {
            sample_rate: 48000,
            channels: 2,
            mic_enabled: false,
            mic_device_index: None,
            mic_volume: 1.0,
            speaker_enabled: false,
            speaker_device_index: None,
            speaker_volume: 1.0,
        }
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ReplayRecordingConfig {
    pub width: u32,
    pub height: u32,
    pub fps: u32,
    pub video_bitrate: u32,
    pub audio: AudioConfig,
    pub buffer_duration_secs: u64,
    pub segment_duration_secs: u64,
    pub show_cursor: bool,
    pub show_border: bool,
    pub ffmpeg_path: String,
    pub temp_path: String,
}

pub fn list_monitors() -> Result<Vec<MonitorInfo>, Box<dyn std::error::Error>> {
    let monitors = Monitor::enumerate()?;

    let monitor_list: Vec<MonitorInfo> = monitors
        .iter()
        .enumerate()
        .map(|(i, mon)| MonitorInfo {
            index: i,
            name: mon.name().unwrap_or_else(|_| format!("Monitor #{}", i + 1)),
            width: mon.width().unwrap_or(0),
            height: mon.height().unwrap_or(0),
            refresh_rate: mon.refresh_rate().unwrap_or(60),
        })
        .collect();

    Ok(monitor_list)
}

pub fn start_replay_buffer(
    monitor_index: usize,
    config: ReplayRecordingConfig,
) -> Result<ReplayBufferHandle, Box<dyn std::error::Error + Send + Sync>> {
    let monitors = Monitor::enumerate()?;

    if monitor_index >= monitors.len() {
        return Err(format!("Invalid monitor index: {}", monitor_index).into());
    }

    let monitor = monitors[monitor_index].clone();

    log::info!("Initializing replay buffer...");
    log::info!(
        "Resolution: {}x{} @ {} FPS",
        config.width, config.height, config.fps
    );
    log::info!(
        "Buffer: {}s (segments: {}s)",
        config.buffer_duration_secs, config.segment_duration_secs
    );

    let replay_buffer = Arc::new(ReplayBuffer::new(ReplayBufferConfig {
        buffer_duration_secs: config.buffer_duration_secs,
        temp_dir: std::path::PathBuf::from(&config.temp_path),
        ffmpeg_path: config.ffmpeg_path.clone(),
    })?);

    let audio_buffer_duration = (config.buffer_duration_secs as f64 * 1.2) as u64;
    let mic_audio_buffer = Arc::new(Mutex::new(CircularAudioBuffer::new(
        Duration::from_secs(audio_buffer_duration),
        config.audio.sample_rate,
        config.audio.channels,
    )));
    let speaker_audio_buffer = Arc::new(Mutex::new(CircularAudioBuffer::new(
        Duration::from_secs(audio_buffer_duration),
        config.audio.sample_rate,
        config.audio.channels,
    )));

    let handle = Arc::new(Mutex::new(ReplayBufferHandle::new(replay_buffer.clone())));
    if let Ok(mut h) = handle.lock() {
        h.set_audio_buffers(mic_audio_buffer.clone(), speaker_audio_buffer.clone());
        h.set_audio_format(config.audio.sample_rate, config.audio.channels);
        h.set_audio_volumes(config.audio.mic_volume, config.audio.speaker_volume);
    }

    log::info!("Replay buffer active");

    let flags = ReplayFlags {
        width: config.width,
        height: config.height,
        fps: config.fps,
        video_bitrate: config.video_bitrate,
        audio_config: config.audio,
        segment_duration_secs: config.segment_duration_secs,
        replay_buffer: replay_buffer.clone(),
        handle: handle.clone(),
        temp_dir: std::path::PathBuf::from(&config.temp_path).join("recording"),
        mic_audio_buffer,
        speaker_audio_buffer,
    };

    std::fs::create_dir_all(&flags.temp_dir)?;

    let cursor_setting = if config.show_cursor {
        CursorCaptureSettings::WithCursor
    } else {
        CursorCaptureSettings::WithoutCursor
    };

    let border_setting = if config.show_border {
        DrawBorderSettings::WithBorder
    } else {
        DrawBorderSettings::WithoutBorder
    };

    let frame_interval = Duration::from_micros(1_000_000 / config.fps as u64);

    let settings = Settings::new(
        monitor,
        cursor_setting,
        border_setting,
        SecondaryWindowSettings::Default,
        MinimumUpdateIntervalSettings::Custom(frame_interval),
        DirtyRegionSettings::Default,
        ColorFormat::Bgra8,
        flags,
    );

    std::thread::spawn(move || {
        if let Err(e) = ReplayScreenRecorder::start(settings) {
            log::error!("Replay buffer error: {:?}", e);
        }
    });

    let h = handle.lock().unwrap().clone();
    Ok(h)
}

struct ReplayFlags {
    width: u32,
    height: u32,
    fps: u32,
    video_bitrate: u32,
    audio_config: AudioConfig,
    segment_duration_secs: u64,
    replay_buffer: Arc<ReplayBuffer>,
    handle: Arc<Mutex<ReplayBufferHandle>>,
    temp_dir: std::path::PathBuf,
    mic_audio_buffer: Arc<Mutex<CircularAudioBuffer>>,
    speaker_audio_buffer: Arc<Mutex<CircularAudioBuffer>>,
}

struct ReplayScreenRecorder {
    encoder: Option<VideoEncoder>,
    current_segment_path: std::path::PathBuf,
    segment_sequence: u64,
    next_segment_sequence: u64,

    mic_capture: Option<AudioCapture>,
    speaker_capture: Option<AudioCapture>,
    mic_audio_buffer: Arc<Mutex<CircularAudioBuffer>>,
    speaker_audio_buffer: Arc<Mutex<CircularAudioBuffer>>,

    segment_wall_start: Instant,
    segment_duration: Duration,
    last_progress: Instant,
    first_frame_received: bool,

    replay_buffer: Arc<ReplayBuffer>,
    handle: Arc<Mutex<ReplayBufferHandle>>,

    width: u32,
    height: u32,
    fps: u32,
    video_bitrate: u32,
    temp_dir: std::path::PathBuf,
    next_encoder: Arc<std::sync::Mutex<Option<VideoEncoder>>>,
    is_preparing_next: bool,

    frames_dropped: u64,
    stop_signal: Arc<AtomicBool>,
}

impl GraphicsCaptureApiHandler for ReplayScreenRecorder {
    type Flags = ReplayFlags;
    type Error = Box<dyn std::error::Error + Send + Sync>;

    fn new(ctx: Context<Self::Flags>) -> Result<Self, Self::Error> {
        let flags = ctx.flags;

        let audio_settings = AudioSettingsBuilder::default().disabled(true);

        let segment_path = flags.temp_dir.join("segment_000000.mp4");
        let encoder = VideoEncoder::new(
            VideoSettingsBuilder::new(flags.width, flags.height)
                .sub_type(VideoSettingsSubType::H264)
                .bitrate(flags.video_bitrate)
                .frame_rate(flags.fps),
            audio_settings,
            ContainerSettingsBuilder::default().sub_type(ContainerSettingsSubType::MPEG4),
            &segment_path,
        )?;

        let init_capture = |enabled: bool, idx: Option<usize>| -> Option<AudioCapture> {
            if enabled {
                let config = AudioCaptureConfig {
                    sample_rate: flags.audio_config.sample_rate,
                    channels: flags.audio_config.channels,
                    enabled: true,
                };
                match AudioCapture::new(idx, config) {
                    Ok(capture) => Some(capture),
                    Err(e) => {
                        log::error!("Failed to init audio source: {}", e);
                        None
                    }
                }
            } else {
                None
            }
        };

        let mic_capture = init_capture(
            flags.audio_config.mic_enabled,
            flags.audio_config.mic_device_index,
        );
        let speaker_capture = init_capture(
            flags.audio_config.speaker_enabled,
            flags.audio_config.speaker_device_index,
        );

        let now = Instant::now();
        let stop_signal = flags.handle.lock().unwrap().stop_signal();

        Ok(Self {
            encoder: Some(encoder),
            current_segment_path: segment_path,
            segment_sequence: 0,
            next_segment_sequence: 1,
            mic_capture,
            speaker_capture,
            mic_audio_buffer: flags.mic_audio_buffer,
            speaker_audio_buffer: flags.speaker_audio_buffer,
            segment_wall_start: now,
            segment_duration: Duration::from_secs(flags.segment_duration_secs),
            last_progress: now,
            first_frame_received: false,
            replay_buffer: flags.replay_buffer,
            handle: flags.handle,
            width: flags.width,
            height: flags.height,
            fps: flags.fps,
            video_bitrate: flags.video_bitrate,
            temp_dir: flags.temp_dir,
            next_encoder: Arc::new(std::sync::Mutex::new(None)),
            is_preparing_next: false,
            frames_dropped: 0,
            stop_signal,
        })
    }

    fn on_frame_arrived(
        &mut self,
        frame: &mut Frame,
        capture_control: InternalCaptureControl,
    ) -> Result<(), Self::Error> {
        if self.stop_signal.load(atomic::Ordering::Relaxed) {
            if let Some(encoder) = self.encoder.take() {
                encoder.finish()?;
            }
            capture_control.stop();
            return Ok(());
        }

        if !self.first_frame_received {
            self.first_frame_received = true;
            self.segment_wall_start = Instant::now();
            log::info!("First frame received, segment timing started");
        }

        if let Some(cap) = &self.mic_capture {
            while let Some(ts_audio) = cap.try_recv() {
                if let Ok(mut buf) = self.mic_audio_buffer.try_lock() {
                    buf.push(&ts_audio.data, ts_audio.callback_time);
                }
            }
        }
        if let Some(cap) = &self.speaker_capture {
            while let Some(ts_audio) = cap.try_recv() {
                if let Ok(mut buf) = self.speaker_audio_buffer.try_lock() {
                    buf.push(&ts_audio.data, ts_audio.callback_time);
                }
            }
        }

        if let Some(encoder) = &mut self.encoder {
            if let Err(e) = encoder.send_frame(frame) {
                log::error!("Failed to send frame: {}", e);
                self.frames_dropped += 1;
            }
        } else {
            self.frames_dropped += 1;
        }

        if self.last_progress.elapsed().as_secs() >= 2 {
            if let Ok(handle) = self.handle.try_lock() {
                let duration = handle.duration();
                let segments = handle.segment_count();
                if self.frames_dropped > 0 {
                    log::debug!(
                        "Buffer: {:.1}s ({} segments) | Dropped: {} frames",
                        duration, segments, self.frames_dropped
                    );
                } else {
                    log::debug!("Buffer: {:.1}s ({} segments)", duration, segments);
                }
            }
            self.last_progress = Instant::now();
        }

        // Segment rotation
        let elapsed = self.segment_wall_start.elapsed();
        if elapsed >= self.segment_duration {
            if self.try_rotate_segment()? {
                self.segment_wall_start = Instant::now();
                self.is_preparing_next = false;
            } else if !self.is_preparing_next {
                self.is_preparing_next = true;
                self.prepare_next_encoder();
            }
        } else if !self.is_preparing_next
            && elapsed + Duration::from_millis(1500) >= self.segment_duration
        {
            self.is_preparing_next = true;
            self.prepare_next_encoder();
        }

        Ok(())
    }

    fn on_closed(&mut self) -> Result<(), Self::Error> {
        self.finalize_current_segment();
        Ok(())
    }
}

impl ReplayScreenRecorder {
    fn prepare_next_encoder(&mut self) {
        let next_encoder_store = self.next_encoder.clone();
        let next_sequence = self.next_segment_sequence;
        let next_path = self
            .temp_dir
            .join(format!("segment_{:06}.mp4", next_sequence));

        let width = self.width;
        let height = self.height;
        let fps = self.fps;
        let bitrate = self.video_bitrate;

        std::thread::spawn(move || {
            let audio_settings = AudioSettingsBuilder::default().disabled(true);

            let encoder_result = VideoEncoder::new(
                VideoSettingsBuilder::new(width, height)
                    .sub_type(VideoSettingsSubType::H264)
                    .bitrate(bitrate)
                    .frame_rate(fps),
                audio_settings,
                ContainerSettingsBuilder::default().sub_type(ContainerSettingsSubType::MPEG4),
                &next_path,
            );

            match encoder_result {
                Ok(encoder) => {
                    let mut store = next_encoder_store.lock().unwrap();
                    *store = Some(encoder);
                }
                Err(e) => {
                    log::error!("Failed to pre-allocate next encoder: {}", e);
                }
            }
        });
    }

    /// Non-blocking segment rotation. Returns true if the swap happened.
    fn try_rotate_segment(&mut self) -> Result<bool, Box<dyn std::error::Error + Send + Sync>> {
        if !self.is_preparing_next {
            return Ok(false);
        }

        let new_encoder = {
            let mut store = self.next_encoder.lock().unwrap();
            match store.take() {
                Some(enc) => enc,
                None => return Ok(false),
            }
        };

        self.segment_sequence += 1;

        let segment_wall_start = self.segment_wall_start;
        let segment_wall_end = Instant::now();

        let new_segment_path = self
            .temp_dir
            .join(format!("segment_{:06}.mp4", self.next_segment_sequence));
        self.next_segment_sequence += 1;

        let old_encoder = self.encoder.replace(new_encoder);
        let old_segment_path = self.current_segment_path.clone();
        self.current_segment_path = new_segment_path;

        let segment_duration = segment_wall_end
            .duration_since(segment_wall_start)
            .as_secs_f64();
        let replay_buffer = self.replay_buffer.clone();

        std::thread::spawn(move || {
            if let Some(encoder) = old_encoder {
                if let Err(e) = encoder.finish() {
                    log::error!("Failed to finish segment: {}", e);
                    return;
                }
            }

            if let Ok(metadata) = std::fs::metadata(&old_segment_path) {
                if metadata.len() > 0 {
                    if let Err(e) = replay_buffer.add_segment(
                        old_segment_path.clone(),
                        segment_duration,
                        segment_wall_start,
                        segment_wall_end,
                    ) {
                        log::error!("Failed to add segment to buffer: {}", e);
                        let _ = std::fs::remove_file(&old_segment_path);
                    }
                } else {
                    let _ = std::fs::remove_file(&old_segment_path);
                }
            }
        });

        Ok(true)
    }

    fn finalize_current_segment(&mut self) {
        let segment_wall_start = self.segment_wall_start;
        let segment_wall_end = Instant::now();

        let old_encoder = self.encoder.take();
        let old_segment_path = self.current_segment_path.clone();

        let segment_duration = segment_wall_end
            .duration_since(segment_wall_start)
            .as_secs_f64();

        if let Some(encoder) = old_encoder {
            if let Err(e) = encoder.finish() {
                log::error!("Failed to finish final segment: {}", e);
                return;
            }
        }

        if let Ok(metadata) = std::fs::metadata(&old_segment_path) {
            if metadata.len() > 0 {
                if let Err(e) = self.replay_buffer.add_segment(
                    old_segment_path.clone(),
                    segment_duration,
                    segment_wall_start,
                    segment_wall_end,
                ) {
                    log::error!("Failed to add final segment: {}", e);
                    let _ = std::fs::remove_file(&old_segment_path);
                }
            }
        }
    }
}
