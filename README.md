
<p align="center">
  <img src="readme/rewind_banner.png" alt="Rewind Banner" width="100%">
</p>


<h1 align="center">Rewind</h1>

<p align="center">
  An elegant screen recording application with instant replay capability, written in Go and Rust, powered by FFmpeg.
</p>

<div align="center">

[![Rust](https://img.shields.io/badge/Rust-000000?logo=rust&logoColor=white)](https://www.rust-lang.org)
[![Golang](https://img.shields.io/badge/Golang-00ADD8?logo=go&logoColor=white)](https://golang.org)
[![FFmpeg](https://img.shields.io/badge/FFmpeg-FF0000?logo=ffmpeg&logoColor=white)](https://ffmpeg.org)
[![Windows](https://custom-icon-badges.demolab.com/badge/Windows-0078D6?logo=windows11&logoColor=white)](#)
[![Wails](https://img.shields.io/badge/Wails-111827?logo=go&logoColor=white)](https://wails.io)
[![Release](https://img.shields.io/badge/Release-v1.0.1-green)](https://github.com/your/repo/releases)


</div>




## Overview

> Rewind continuously captures your screen in the background and lets you save the last moments on demand. Built with Rust for high-performance video capture and Go for application logic.
>
> *Co-authored with AI via vibe coding — where intuition meets infrastructure.*

## Features

- **Instant Replay** - Capture the last N seconds of screen activity with a single keystroke
- **Hardware Acceleration** - Native Windows Graphics Capture API with GPU encoding (H264)
- **Audio Capture** - Record system audio and microphone simultaneously with independent volume controls
- **Device Monitoring** - Automatically detects and handles monitor/audio device changes without polling
- **System Tray Integration** - Runs silently in the background with quick access via tray icon
- **Global Hotkeys** - Control recording without leaving your current application
- **Configurable Quality** - Adjustable FPS, bitrate, and buffer duration
- **Modern Interface** - Clean, frameless UI with smooth animations


## Tech Stack

- **Rust** - High-performance native video/audio capture using Windows Graphics Capture API
- **Go** - Application logic and orchestration
- **Wails v3** - Modern desktop application framework
- **FFmpeg** - Video segment concatenation and audio muxing
- **CPAL** - Cross-platform audio library for microphone and loopback capture
- **React** - Dynamic and responsive frontend user interface
- **TypeScript** - Type-safe development for robust code



## Installation

### Windows

1. **Download** the latest release from the [Releases](https://github.com/emirakts0/rewind/releases) page.     
2. **Choose** your installation method:
   - **Setup Installer** (`RewindSetup.exe`) - Installs to Program Files with shortcuts
   - **Portable Version** (`Rewind-Portable.zip`) - Extract and run from anywhere
3. **Run** the application and it will start minimized to the system tray

## Getting Started

1. **Launch** Rewind - it will appear in your system tray
2. **Start Recording** with <kbd>Ctrl</kbd> + <kbd>F9</kbd> to begin buffering
3. **Capture Moments** with <kbd>Ctrl</kbd> + <kbd>F10</kbd> to save the last N seconds
4. **Find Your Clips** in the clips folder (default: `%APPDATA%\Rewind\clips`)

### Configuration

- **Buffer Duration**: Set how many seconds to keep in memory (default: 30s)
- **Video Quality**: Adjust FPS (30/60/..) and bitrate.
- **Audio Sources**: Enable/disable system audio and microphone
- **Output Location**: Choose where clips are saved
- **Hardware Encoder**: Select your preferred GPU encoder or use cpu encoding


## Screenshots

<div style="display: flex; justify-content: center; gap: 10px; flex-wrap: wrap;">
  <img src="readme/img.png"   width="270">
  <img src="readme/img_1.png" width="270">
  <img src="readme/img_2.png" width="270">
  <img src="readme/img_3.png" width="270">
  <img src="readme/img_4.png" width="270">
</div>


## Building from Source

### Prerequisites

```bash
    # Install Rust (latest stable)
    # Install Go (1.21+)
    # Install Node.js (18+)
    # Install Wails CLI
    go install github.com/wailsapp/wails/v3/cmd/wails3@latest
    # Install NSIS for creating Windows installer
    # Download from https://nsis.sourceforge.io/
    # Download FFmpeg from the Releases page and place it in the bin/ folder.
```

### Build Steps

```bash
  # Build everything (executable + installer + portable ZIP)
  wails3 task package
```

### Development Mode (with Hot Reload)


```bash
  wails3 task dev
```

## Architecture

Rewind uses a circular buffer approach with native Windows capture:

- **Rust Capture Engine**: Uses Windows Graphics Capture API for zero-copy GPU frame capture
- **Video Encoder**: Hardware-accelerated H264 encoding via Windows Media Foundation
- **Audio Capture**: CPAL-based microphone and loopback recording with circular buffers
- **Device Monitor**: Windows WM_DEVICECHANGE message loop for real-time device change detection
- **Segment Manager**: Maintains rolling window of N-second video segments
- **Go Application Layer**: Orchestrates capture, manages UI, and handles user interactions
- **FFmpeg Integration**: Concatenates segments and muxes audio on save

### Device Change Handling

The application monitors device changes in real-time using Windows device notifications:

- **Idle State**: When a monitor or audio device is removed, the configuration is automatically validated and updated
- **Recording State**: If the currently recording monitor or audio device is disconnected, recording stops automatically with a notification
- **No Polling**: Uses native Windows `WM_DEVICECHANGE` messages for efficient, event-driven monitoring
- **Frontend Updates**: Device list changes trigger automatic UI refresh

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

---


<p align="center">
  <img src="build/assets/icon.ico" alt="Rewind Icon" width="30" style="vertical-align: middle; margin-bottom: 4px;">
  <a href="mailto:emirakts0@gmail.com">emirakts0@gmail.com</a>
</p>
