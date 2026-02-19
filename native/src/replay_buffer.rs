use std::collections::VecDeque;
use std::fs::{self, File};
use std::io::Write;
use std::path::{Path, PathBuf};
use std::sync::atomic::{self, AtomicBool};
use std::sync::{Arc, Mutex};
use std::time::Instant;

use crate::audio_capture::CircularAudioBuffer;

#[derive(Debug, Clone)]
pub struct ReplayBufferConfig {
    pub buffer_duration_secs: u64,
    pub temp_dir: PathBuf,
    pub ffmpeg_path: String,
}

impl Default for ReplayBufferConfig {
    fn default() -> Self {
        Self {
            buffer_duration_secs: 40,
            temp_dir: PathBuf::from("./replay_temp"),
            ffmpeg_path: "ffmpeg".into(),
        }
    }
}

#[derive(Debug, Clone)]
struct SegmentInfo {
    path: PathBuf,
    duration_secs: f64,
    wall_start: Instant,
    wall_end: Instant,
    file_size: u64,
}

pub struct ReplayBuffer {
    config: ReplayBufferConfig,
    segments: Arc<Mutex<VecDeque<SegmentInfo>>>,
    next_sequence: Arc<Mutex<u64>>,
    total_duration: Arc<Mutex<f64>>,
    total_disk_usage: Arc<Mutex<u64>>,
}

impl ReplayBuffer {
    pub fn new(
        config: ReplayBufferConfig,
    ) -> Result<Self, Box<dyn std::error::Error + Send + Sync>> {
        fs::create_dir_all(&config.temp_dir)?;
        Self::cleanup_temp_dir(&config.temp_dir)?;

        Ok(Self {
            config,
            segments: Arc::new(Mutex::new(VecDeque::new())),
            next_sequence: Arc::new(Mutex::new(0)),
            total_duration: Arc::new(Mutex::new(0.0)),
            total_disk_usage: Arc::new(Mutex::new(0)),
        })
    }

    pub fn add_segment(
        &self,
        source_path: PathBuf,
        duration_secs: f64,
        wall_start: Instant,
        wall_end: Instant,
    ) -> Result<(), Box<dyn std::error::Error + Send + Sync>> {
        let mut segments = self.segments.lock().unwrap();
        let mut next_seq = self.next_sequence.lock().unwrap();
        let mut total_dur = self.total_duration.lock().unwrap();
        let mut total_disk = self.total_disk_usage.lock().unwrap();

        let sequence = *next_seq;
        *next_seq += 1;

        let dest_path = self
            .config
            .temp_dir
            .join(format!("segment_{:06}.mp4", sequence));

        let mut success = false;
        for _ in 0..5 {
            if let Err(_) = fs::rename(&source_path, &dest_path) {
                std::thread::sleep(std::time::Duration::from_millis(50));
            } else {
                success = true;
                break;
            }
        }

        if !success {
            fs::copy(&source_path, &dest_path)?;
            let _ = fs::remove_file(&source_path);
        }

        let file_size = fs::metadata(&dest_path)
            .map(|m| m.len())
            .unwrap_or(0);

        segments.push_back(SegmentInfo {
            path: dest_path,
            duration_secs,
            wall_start,
            wall_end,
            file_size,
        });

        *total_dur += duration_secs;
        *total_disk += file_size;

        while *total_dur > self.config.buffer_duration_secs as f64 {
            if let Some(old_segment) = segments.pop_front() {
                *total_dur -= old_segment.duration_secs;
                *total_disk = total_disk.saturating_sub(old_segment.file_size);
                let _ = fs::remove_file(&old_segment.path);
            } else {
                break;
            }
        }

        Ok(())
    }

    pub fn save_to_file<P: AsRef<Path>>(
        &self,
        output_path: P,
        mic_audio_buffer: Option<&Arc<Mutex<CircularAudioBuffer>>>,
        speaker_audio_buffer: Option<&Arc<Mutex<CircularAudioBuffer>>>,
        audio_sample_rate: u32,
        audio_channels: u16,
        mic_volume: f32,
        speaker_volume: f32,
    ) -> Result<(), Box<dyn std::error::Error + Send + Sync>> {
        let timeout_duration = std::time::Duration::from_secs(45);
        let start_time = Instant::now();
        
        let result = self.save_to_file_internal(
            output_path,
            mic_audio_buffer,
            speaker_audio_buffer,
            audio_sample_rate,
            audio_channels,
            mic_volume,
            speaker_volume,
            start_time,
            timeout_duration,
        );
        
        result
    }

    fn save_to_file_internal<P: AsRef<Path>>(
        &self,
        output_path: P,
        mic_audio_buffer: Option<&Arc<Mutex<CircularAudioBuffer>>>,
        speaker_audio_buffer: Option<&Arc<Mutex<CircularAudioBuffer>>>,
        audio_sample_rate: u32,
        audio_channels: u16,
        mic_volume: f32,
        speaker_volume: f32,
        start_time: Instant,
        timeout_duration: std::time::Duration,
    ) -> Result<(), Box<dyn std::error::Error + Send + Sync>> {
        let (segment_paths, video_wall_start, video_wall_end, seg_count, total_dur) = {
            let segments = self.segments.lock().unwrap();

            if segments.is_empty() {
                return Err("No segments available to save".into());
            }

            let paths: Vec<PathBuf> = segments.iter().map(|s| s.path.clone()).collect();
            let wall_start = segments.front().unwrap().wall_start;
            let wall_end = segments.back().unwrap().wall_end;
            let count = segments.len();
            let dur = *self.total_duration.lock().unwrap();

            (paths, wall_start, wall_end, count, dur)
        };

        log::info!("Saving replay buffer...");
        log::info!("Segments: {}", seg_count);
        log::info!("Duration: {:.1}s", total_dur);

        let video_wall_duration = video_wall_end
            .duration_since(video_wall_start)
            .as_secs_f64();
        log::info!("Video wall-clock span: {:.2}s", video_wall_duration);

        let output_path = output_path.as_ref();

        log::info!("Preparing video concat list...");
        let concat_list_path = self.config.temp_dir.join("concat_list.txt");
        let mut concat_file = File::create(&concat_list_path)?;

        for path in &segment_paths {
            let abs_path = std::fs::canonicalize(path)?;
            let path_str = abs_path.to_string_lossy();

            let clean_path = if path_str.starts_with(r"\\?\") {
                &path_str[4..]
            } else {
                &path_str
            };
            
            writeln!(concat_file, "file '{}'", clean_path)?;
        }
        concat_file.sync_all()?;

        if start_time.elapsed() >= timeout_duration {
            let _ = std::fs::remove_file(&concat_list_path);
            return Err("Save operation timed out (45s limit exceeded)".into());
        }

        // Calculate video duration from segments
        let video_duration = total_dur;

        let bpf = (audio_channels as usize) * 2;

        let mic_data = mic_audio_buffer.and_then(|buf| {
            let buffer = buf.lock().ok()?;
            let data = buffer.get_samples_from(video_wall_start);
            if data.is_empty() {
                None
            } else {
                Some(data)
            }
        });

        let speaker_data = speaker_audio_buffer.and_then(|buf| {
            let buffer = buf.lock().ok()?;
            let data = buffer.get_samples_from(video_wall_start);
            if data.is_empty() {
                None
            } else {
                Some(data)
            }
        });

        let has_audio = mic_data.is_some() || speaker_data.is_some();

        if has_audio {
            log::info!("Processing audio...");

            if let Some(ref d) = mic_data {
                let dur = (d.len() / bpf) as f64 / audio_sample_rate as f64;
                log::info!("Mic audio: {:.2}s", dur);
            }
            if let Some(ref d) = speaker_data {
                let dur = (d.len() / bpf) as f64 / audio_sample_rate as f64;
                log::info!("Speaker audio: {:.2}s", dur);
            }

            // Helper function to apply volume to audio data
            let apply_volume = |data: &[u8], volume: f32| -> Vec<u8> {
                let frames = data.len() / bpf;
                let mut processed = Vec::with_capacity(data.len());
                
                for i in 0..frames {
                    let offset = i * bpf;
                    for ch in 0..(audio_channels as usize) {
                        let byte_off = offset + ch * 2;
                        let sample = i16::from_le_bytes([data[byte_off], data[byte_off + 1]]) as f32;
                        let adjusted = (sample * volume).clamp(-32768.0, 32767.0) as i16;
                        processed.extend_from_slice(&adjusted.to_le_bytes());
                    }
                }
                processed
            };

            let audio_data = match (&mic_data, &speaker_data) {
                (Some(mic), Some(speaker)) => {
                    let max_len = mic.len().max(speaker.len());
                    let max_len = (max_len / bpf) * bpf;
                    let max_frames = max_len / bpf;

                    let mut mixed = Vec::with_capacity(max_len);

                    for i in 0..max_frames {
                        let offset = i * bpf;
                        for ch in 0..(audio_channels as usize) {
                            let byte_off = offset + ch * 2;

                            let mic_sample = if byte_off + 1 < mic.len() {
                                i16::from_le_bytes([mic[byte_off], mic[byte_off + 1]]) as f32
                            } else {
                                0.0
                            };
                            let speaker_sample = if byte_off + 1 < speaker.len() {
                                i16::from_le_bytes([speaker[byte_off], speaker[byte_off + 1]])
                                    as f32
                            } else {
                                0.0
                            };

                            let mixed_sample = (mic_sample * mic_volume + speaker_sample * speaker_volume)
                                .clamp(-32768.0, 32767.0)
                                as i16;
                            mixed.extend_from_slice(&mixed_sample.to_le_bytes());
                        }
                    }

                    log::info!(
                        "Mixed mic + speaker: {:.2}s",
                        (mixed.len() / bpf) as f64 / audio_sample_rate as f64
                    );
                    mixed
                }
                (Some(mic), None) => apply_volume(mic, mic_volume),
                (None, Some(speaker)) => apply_volume(speaker, speaker_volume),
                (None, None) => unreachable!(),
            };

            // Trim/pad audio to match video duration
            let target_samples = (video_duration * audio_sample_rate as f64) as usize;
            let target_bytes = target_samples * bpf;

            let final_audio = if audio_data.len() < target_bytes {
                let deficit = target_bytes - audio_data.len();
                log::info!(
                    "Padding {:.3}s silence at end",
                    (deficit / bpf) as f64 / audio_sample_rate as f64
                );
                let mut padded = audio_data;
                padded.resize(target_bytes, 0);
                padded
            } else if audio_data.len() > target_bytes {
                let excess = audio_data.len() - target_bytes;
                log::info!(
                    "Trimming {:.3}s from end to match video",
                    (excess / bpf) as f64 / audio_sample_rate as f64
                );
                let mut trimmed = audio_data;
                trimmed.truncate(target_bytes);
                trimmed
            } else {
                audio_data
            };

            let final_dur = (final_audio.len() / bpf) as f64 / audio_sample_rate as f64;
            log::info!(
                "Final audio: {:.2}s → matches video {:.2}s",
                final_dur, video_duration
            );

            let temp_audio = self.config.temp_dir.join("temp_audio.pcm");
            let mut audio_file = File::create(&temp_audio)?;
            audio_file.write_all(&final_audio)?;
            audio_file.sync_all()?;

            if start_time.elapsed() >= timeout_duration {
                let _ = std::fs::remove_file(&temp_audio);
                let _ = std::fs::remove_file(&concat_list_path);
                return Err("Save operation timed out before muxing".into());
            }

            log::info!("Concatenating video + muxing audio in single FFmpeg pass...");

            let ar_str = audio_sample_rate.to_string();
            let ac_str = audio_channels.to_string();

            let output = std::process::Command::new(&self.config.ffmpeg_path)
                .args(&[
                    "-f",
                    "concat",
                    "-safe",
                    "0",
                    "-i",
                    concat_list_path.to_str().unwrap(),
                    "-f",
                    "s16le",
                    "-ar",
                    &ar_str,
                    "-ac",
                    &ac_str,
                    "-i",
                    temp_audio.to_str().unwrap(),
                    "-c:v",
                    "copy",
                    "-c:a",
                    "aac",
                    "-b:a",
                    "192k",
                    "-shortest",
                    "-map",
                    "0:v:0",
                    "-map",
                    "1:a:0",
                    "-y",
                    output_path.to_str().unwrap(),
                ])
                .output()?;

            if !output.stderr.is_empty() {
                for line in String::from_utf8_lossy(&output.stderr).lines() {
                    if !line.trim().is_empty() {
                        log::debug!("[FFMPEG] {}", line);
                    }
                }
            }

            let _ = std::fs::remove_file(&temp_audio);
            let _ = std::fs::remove_file(&concat_list_path);

            if !output.status.success() {
                return Err("FFmpeg concat+mux failed".into());
            }

            if start_time.elapsed() >= timeout_duration {
                let _ = std::fs::remove_file(output_path);
                return Err("Save operation timed out after muxing".into());
            }
        } else {
            log::info!("No audio data available, concatenating video only...");
            
            if start_time.elapsed() >= timeout_duration {
                let _ = std::fs::remove_file(&concat_list_path);
                return Err("Save operation timed out (45s limit exceeded)".into());
            }
            
            let output = std::process::Command::new(&self.config.ffmpeg_path)
                .args(&[
                    "-f",
                    "concat",
                    "-safe",
                    "0",
                    "-i",
                    concat_list_path.to_str().unwrap(),
                    "-c",
                    "copy",
                    "-y",
                    output_path.to_str().unwrap(),
                ])
                .output()?;

            if !output.stderr.is_empty() {
                for line in String::from_utf8_lossy(&output.stderr).lines() {
                    if !line.trim().is_empty() {
                        log::debug!("[FFMPEG] {}", line);
                    }
                }
            }

            let _ = std::fs::remove_file(&concat_list_path);

            if !output.status.success() {
                return Err("FFmpeg video concat failed".into());
            }
        }

        log::info!("Saved to: {}", output_path.display());

        self.clear()?;
        log::info!("   Video segments cleared, ready for new recording");

        Ok(())
    }

    fn cleanup_temp_dir(dir: &Path) -> Result<(), Box<dyn std::error::Error + Send + Sync>> {
        if dir.exists() {
            for entry in fs::read_dir(dir)? {
                let entry = entry?;
                let path = entry.path();
                if path.is_file() {
                    let ext = path.extension().and_then(|e| e.to_str());
                    if ext == Some("mp4")
                        || ext == Some("txt")
                        || ext == Some("ts")
                        || ext == Some("pcm")
                    {
                        let _ = fs::remove_file(path);
                    }
                }
            }
        }
        Ok(())
    }

    pub fn clear(&self) -> Result<(), Box<dyn std::error::Error + Send + Sync>> {
        let mut segments = self.segments.lock().unwrap();
        let mut total_dur = self.total_duration.lock().unwrap();
        let mut total_disk = self.total_disk_usage.lock().unwrap();

        for segment in segments.drain(..) {
            let _ = fs::remove_file(&segment.path);
        }

        *total_dur = 0.0;
        *total_disk = 0;

        Ok(())
    }

    pub fn current_duration(&self) -> f64 {
        *self.total_duration.lock().unwrap()
    }

    pub fn segment_count(&self) -> usize {
        self.segments.lock().unwrap().len()
    }

    pub fn disk_usage(&self) -> u64 {
        *self.total_disk_usage.lock().unwrap()
    }
}

impl Drop for ReplayBuffer {
    fn drop(&mut self) {
        let _ = self.clear();
    }
}

#[derive(Clone)]
pub struct ReplayBufferHandle {
    buffer: Arc<ReplayBuffer>,
    stop_signal: Arc<AtomicBool>,
    is_saving: Arc<AtomicBool>,
    mic_audio_buffer: Option<Arc<Mutex<CircularAudioBuffer>>>,
    speaker_audio_buffer: Option<Arc<Mutex<CircularAudioBuffer>>>,
    audio_sample_rate: u32,
    audio_channels: u16,
    mic_volume: f32,
    speaker_volume: f32,
}

impl ReplayBufferHandle {
    pub fn new(buffer: Arc<ReplayBuffer>) -> Self {
        Self {
            buffer,
            stop_signal: Arc::new(AtomicBool::new(false)),
            is_saving: Arc::new(AtomicBool::new(false)),
            mic_audio_buffer: None,
            speaker_audio_buffer: None,
            audio_sample_rate: 48000,
            audio_channels: 2,
            mic_volume: 1.0,
            speaker_volume: 1.0,
        }
    }

    pub fn set_audio_buffers(
        &mut self,
        mic_buffer: Arc<Mutex<CircularAudioBuffer>>,
        speaker_buffer: Arc<Mutex<CircularAudioBuffer>>,
    ) {
        self.mic_audio_buffer = Some(mic_buffer);
        self.speaker_audio_buffer = Some(speaker_buffer);
    }

    pub fn set_audio_format(&mut self, sample_rate: u32, channels: u16) {
        self.audio_sample_rate = sample_rate;
        self.audio_channels = channels;
    }

    pub fn set_audio_volumes(&mut self, mic_volume: f32, speaker_volume: f32) {
        self.mic_volume = mic_volume;
        self.speaker_volume = speaker_volume;
    }

    pub fn save<P: AsRef<Path>>(
        &self,
        output_path: P,
    ) -> Result<(), Box<dyn std::error::Error + Send + Sync>> {
        if self.is_saving.swap(true, atomic::Ordering::SeqCst) {
            return Err("Save operation already in progress".into());
        }

        let result = self.buffer.save_to_file(
            output_path,
            self.mic_audio_buffer.as_ref(),
            self.speaker_audio_buffer.as_ref(),
            self.audio_sample_rate,
            self.audio_channels,
            self.mic_volume,
            self.speaker_volume,
        );

        self.is_saving.store(false, atomic::Ordering::SeqCst);
        result
    }

    pub fn duration(&self) -> f64 {
        self.buffer.current_duration()
    }

    pub fn segment_count(&self) -> usize {
        self.buffer.segment_count()
    }

    pub fn stop(&self) {
        self.stop_signal.store(true, atomic::Ordering::Relaxed);
    }

    pub fn clear(&self) -> Result<(), Box<dyn std::error::Error + Send + Sync>> {
        if self.is_saving.load(atomic::Ordering::SeqCst) {
            return Err("Cannot clear segments while saving".into());
        }
        self.buffer.clear()
    }

    pub fn stop_signal(&self) -> Arc<AtomicBool> {
        self.stop_signal.clone()
    }

    pub fn is_saving(&self) -> bool {
        self.is_saving.load(atomic::Ordering::SeqCst)
    }

    pub fn disk_usage_bytes(&self) -> u64 {
        self.buffer.disk_usage()
    }

    pub fn memory_usage_bytes(&self) -> u64 {
        let mut total = 0u64;

        if let Some(ref mic_buf) = self.mic_audio_buffer {
            if let Ok(buf) = mic_buf.lock() {
                total += buf.current_bytes() as u64;
            }
        }

        if let Some(ref speaker_buf) = self.speaker_audio_buffer {
            if let Ok(buf) = speaker_buf.lock() {
                total += buf.current_bytes() as u64;
            }
        }

        total
    }
}
