$ErrorActionPreference = "Stop"

Write-Host "Starting Rewind Build Process..." -ForeColor Cyan

# Path to NSIS (Auto-detected + Common Path)
$NSISPath = "makensis"
if (Test-Path "C:\Program Files (x86)\NSIS\makensis.exe") {
    $NSISPath = "C:\Program Files (x86)\NSIS\makensis.exe"
    Write-Host "Found NSIS at: $NSISPath" -ForeColor Green
}

# 1. Build Frontend
Write-Host "Building Frontend..." -ForeColor Yellow
Push-Location frontend
try {
    npm install
    npm run build
}
catch {
    Write-Error "Frontend build failed"
    exit 1
}
finally {
    Pop-Location
}

# 2. Prepare Build Directory
Write-Host "Preparing Build Directory..." -ForeColor Yellow
if (!(Test-Path "build")) { New-Item -ItemType Directory -Force -Path "build" }
if (!(Test-Path "build/bin")) { New-Item -ItemType Directory -Force -Path "build/bin" }

# 3. Copy Sidecar (FFmpeg)
Write-Host "Copying FFmpeg Sidecar..." -ForeColor Yellow
if (Test-Path "bin/ffmpeg.exe") {
    Copy-Item "bin/ffmpeg.exe" -Destination "build/bin/ffmpeg.exe" -Force
} else {
    Write-Error "bin/ffmpeg.exe not found! Please place ffmpeg.exe in the 'bin' folder at the project root."
    exit 1
}

# 4. Build Backend (Go)
Write-Host "Building Backend (Go)..." -ForeColor Yellow
$ldflags = "-H windowsgui -s -w" # Hide console, strip symbols
try {
    go build -tags windows -ldflags $ldflags -o build/rewind.exe .
}
catch {
    Write-Error "Go build failed"
    exit 1
}

# 5. Create Installer (NSIS)
Write-Host "Creating Installer..." -ForeColor Yellow

try {
    # Run NSIS on the file in root directory which points to build/ folder
    & $NSISPath installer.nsi
    
    if ($LASTEXITCODE -eq 0) {
        Write-Host "Installer created successfully: build/RewindSetup.exe" -ForeColor Green
    } else {
        throw "NSIS exited with code $LASTEXITCODE"
    }
}
catch {
    Write-Warning "NSIS build failed: $_"
    
    # Create a portable zip just in case
    Write-Host "Creating portable Zip archive as fallback..." -ForeColor Cyan
    Compress-Archive -Path "build/rewind.exe", "build/bin" -DestinationPath "build/Rewind-Portable.zip" -Force
    Write-Host "Portable Zip created: build/Rewind-Portable.zip" -ForeColor Green
}
