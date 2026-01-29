; Rewind Installer Script
; Requires NSIS 3.0+

!include "MUI2.nsh"

; General
Name "Rewind"
OutFile "RewindSetup.exe"
InstallDir "$PROGRAMFILES64\Rewind"
InstallDirRegKey HKCU "Software\Rewind" ""
RequestExecutionLevel admin

; UI Interface
!define MUI_ABORTWARNING
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH

!insertmacro MUI_LANGUAGE "English"

Section "Rewind" SecRewind
    SetOutPath "$INSTDIR"
    
    ; Main executable
    File "rewind.exe"
    
    ; FFmpeg Sidecar
    SetOutPath "$INSTDIR\bin"
    File "bin\ffmpeg.exe"
    
    ; Restore path to root for uninstaller
    SetOutPath "$INSTDIR"
    
    ; Uninstaller
    WriteUninstaller "$INSTDIR\uninstall.exe"
    
    ; Registry keys
    WriteRegStr HKCU "Software\Rewind" "" $INSTDIR
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\Rewind" "DisplayName" "Rewind"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\Rewind" "UninstallString" "$\"$INSTDIR\uninstall.exe$\""
SectionEnd

Section "Desktop Shortcut" SecShortcut
    CreateShortcut "$DESKTOP\Rewind.lnk" "$INSTDIR\rewind.exe"
SectionEnd

Section "Uninstall"
    Delete "$INSTDIR\rewind.exe"
    Delete "$INSTDIR\uninstall.exe"
    Delete "$INSTDIR\bin\ffmpeg.exe"
    RMDir "$INSTDIR\bin"
    RMDir "$INSTDIR"
    Delete "$DESKTOP\Rewind.lnk"
    DeleteRegKey HKCU "Software\Rewind"
    DeleteRegKey HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\Rewind"
SectionEnd
