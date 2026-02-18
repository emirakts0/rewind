use serde::{Deserialize, Serialize};
use std::collections::VecDeque;
use std::sync::mpsc::{self, Receiver};
use std::sync::{Arc, Mutex};
use std::time::{Duration, Instant};

use cpal::traits::{DeviceTrait, HostTrait, StreamTrait};
use cpal::{Device, Host, Stream, StreamConfig};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AudioDeviceInfo {
    pub index: usize,
    pub name: String,
    pub is_input: bool,
}

#[derive(Debug, Clone)]
pub struct AudioCaptureConfig {
    pub sample_rate: u32,
    pub channels: u16,
    pub enabled: bool,
}

impl Default for AudioCaptureConfig {
    fn default() -> Self {
        Self {
            sample_rate: 48000,
            channels: 2,
            enabled: false,
        }
    }
}

pub fn list_audio_devices() -> Result<Vec<AudioDeviceInfo>, Box<dyn std::error::Error>> {
    let host = cpal::default_host();
    let mut devices = Vec::new();
    let mut i = 0;

    if let Ok(inputs) = host.input_devices() {
        for device in inputs {
            if let Ok(desc) = device.description() {
                devices.push(AudioDeviceInfo {
                    index: i,
                    name: desc.name().to_string(),
                    is_input: true,
                });
                i += 1;
            }
        }
    }

    if let Ok(outputs) = host.output_devices() {
        for device in outputs {
            if let Ok(desc) = device.description() {
                devices.push(AudioDeviceInfo {
                    index: i,
                    name: format!("{} [Loopback]", desc.name()),
                    is_input: false,
                });
                i += 1;
            }
        }
    }

    Ok(devices)
}

pub struct TimestampedAudio {
    pub data: Vec<u8>,
    pub callback_time: Instant,
}

pub struct AudioCapture {
    _stream: Stream,
    receiver: Arc<Mutex<Receiver<TimestampedAudio>>>,
}

impl AudioCapture {
    pub fn new(
        device_index: Option<usize>,
        config: AudioCaptureConfig,
    ) -> Result<Self, Box<dyn std::error::Error>> {
        if !config.enabled {
            return Err("Audio capture is disabled".into());
        }

        let host = cpal::default_host();
        let (device, is_input) = if let Some(idx) = device_index {
            Self::find_device_by_index(&host, idx)?
        } else {
            let device = host
                .default_input_device()
                .ok_or("No default input device")?;
            (device, true)
        };

        if let Ok(desc) = device.description() {
            log::info!("Using audio device: {}", desc.name());
        }

        let stream_config = StreamConfig {
            channels: config.channels,
            sample_rate: config.sample_rate,
            buffer_size: cpal::BufferSize::Default,
        };

        let (sender, receiver) = mpsc::channel::<TimestampedAudio>();
        let sender = Arc::new(Mutex::new(sender));

        let err_fn = |err| log::error!("Audio stream error: {}", err);

        let stream = device.build_input_stream(
            &stream_config,
            move |data: &[f32], _: &cpal::InputCallbackInfo| {
                let callback_time = Instant::now();

                let pcm_data: Vec<i16> = data
                    .iter()
                    .map(|&sample| (sample.clamp(-1.0, 1.0) * 32767.0) as i16)
                    .collect();

                let bytes: Vec<u8> = pcm_data
                    .iter()
                    .flat_map(|&sample| sample.to_le_bytes())
                    .collect();

                if let Ok(sender) = sender.lock() {
                    let _ = sender.send(TimestampedAudio {
                        data: bytes,
                        callback_time,
                    });
                }
            },
            err_fn,
            None,
        )?;

        stream.play()?;

        log::info!(
            "Audio capture started: {}Hz, {} channels ({})",
            config.sample_rate,
            config.channels,
            if is_input { "Input" } else { "Loopback" }
        );

        Ok(Self {
            _stream: stream,
            receiver: Arc::new(Mutex::new(receiver)),
        })
    }

    pub fn try_recv(&self) -> Option<TimestampedAudio> {
        self.receiver.lock().ok()?.try_recv().ok()
    }

    fn find_device_by_index(
        host: &Host,
        target_index: usize,
    ) -> Result<(Device, bool), Box<dyn std::error::Error>> {
        let mut idx = 0;

        if let Ok(inputs) = host.input_devices() {
            for device in inputs {
                if idx == target_index {
                    return Ok((device, true));
                }
                idx += 1;
            }
        }

        if let Ok(outputs) = host.output_devices() {
            for device in outputs {
                if device.description().is_ok() {
                    if idx == target_index {
                        return Ok((device, false));
                    }
                    idx += 1;
                }
            }
        }

        Err(format!("Audio device index {} not found", target_index).into())
    }
}

#[derive(Clone)]
struct AudioChunk {
    data: Vec<u8>,
    wall_time: Instant,
}

pub struct CircularAudioBuffer {
    chunks: VecDeque<AudioChunk>,
    max_bytes: usize,
    current_bytes: usize,
    sample_rate: u32,
    channels: u16,
}

impl CircularAudioBuffer {
    pub fn new(max_duration: Duration, sample_rate: u32, channels: u16) -> Self {
        let max_bytes =
            (max_duration.as_secs_f64() * sample_rate as f64 * channels as f64 * 2.0) as usize;

        log::info!(
            "Audio buffer: {:.1}s capacity ({:.2} MB)",
            max_duration.as_secs_f64(),
            max_bytes as f64 / 1_048_576.0
        );

        Self {
            chunks: VecDeque::new(),
            max_bytes,
            current_bytes: 0,
            sample_rate,
            channels,
        }
    }

    fn bytes_per_frame(&self) -> usize {
        (self.channels as usize) * 2
    }

    pub fn push(&mut self, data: &[u8], wall_time: Instant) {
        if data.is_empty() {
            return;
        }

        let bpf = self.bytes_per_frame();
        let aligned_len = (data.len() / bpf) * bpf;
        if aligned_len == 0 {
            return;
        }

        self.current_bytes += aligned_len;
        self.chunks.push_back(AudioChunk {
            data: data[..aligned_len].to_vec(),
            wall_time,
        });

        while self.current_bytes > self.max_bytes {
            if let Some(old) = self.chunks.pop_front() {
                self.current_bytes -= old.data.len();
            } else {
                break;
            }
        }
    }

    /// Extract audio samples from the given wall-clock start time onward.
    /// Aligns audio start with video start for A/V sync.
    pub fn get_samples_from(&self, start: Instant) -> Vec<u8> {
        if self.chunks.is_empty() {
            return Vec::new();
        }

        let bpf = self.bytes_per_frame();
        let mut result = Vec::new();
        let mut started = false;

        for chunk in &self.chunks {
            let chunk_frames = chunk.data.len() / bpf;
            let chunk_duration_secs = chunk_frames as f64 / self.sample_rate as f64;
            let chunk_end_time = chunk.wall_time + Duration::from_secs_f64(chunk_duration_secs);

            if chunk_end_time <= start {
                continue;
            }

            if !started {
                started = true;
                if chunk.wall_time < start {
                    let offset_secs = start.duration_since(chunk.wall_time).as_secs_f64();
                    let offset_frames = (offset_secs * self.sample_rate as f64) as usize;
                    let offset_bytes = ((offset_frames * bpf) / bpf) * bpf;
                    let offset_bytes = offset_bytes.min(chunk.data.len());
                    result.extend_from_slice(&chunk.data[offset_bytes..]);
                } else {
                    result.extend_from_slice(&chunk.data);
                }
            } else {
                result.extend_from_slice(&chunk.data);
            }
        }

        let aligned = (result.len() / bpf) * bpf;
        result.truncate(aligned);
        result
    }

    /// Returns current bytes stored in the buffer
    pub fn current_bytes(&self) -> usize {
        self.current_bytes
    }
}
