# Building Minimal FFmpeg for Windows on Linux

This document describes how to build a minimal `ffmpeg.exe` for Windows on a Linux machine. The resulting binary is ~6MB and contains only the components required by this application. Place it in the bin/ folder of the project.

## Requirements

A Linux machine (Ubuntu/Debian) with internet access.

## Steps

### 1. Install dependencies

```bash
  sudo apt update && sudo apt install -y git make mingw-w64 build-essential pkg-config nasm
```

### 2. Clone FFmpeg source

```bash
    git clone https://git.ffmpeg.org/ffmpeg.git --depth=1
    cd ffmpeg
```

### 3. Configure

```bash
    ./configure \
      --cross-prefix=x86_64-w64-mingw32- \
      --arch=x86_64 \
      --target-os=mingw32 \
      --disable-everything \
      --enable-protocol=file \
      --enable-demuxer=concat,mov,pcm_s16le \
      --enable-muxer=mp4 \
      --enable-decoder=h264,aac,pcm_s16le \
      --enable-encoder=aac \
      --enable-parser=h264,aac \
      --enable-bsf=aac_adtstoasc,h264_mp4toannexb \
      --enable-filter=aresample \
      --enable-static \
      --disable-shared \
      --disable-doc \
      --disable-ffplay \
      --disable-ffprobe \
      --disable-network \
      --disable-autodetect
```

### 4. Build

```bash
  make -j$(nproc)
```
