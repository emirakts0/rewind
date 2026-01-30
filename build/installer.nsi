; Rewind Installer Script
; Requires NSIS 3.0+

!include "MUI2.nsh"
!include "LogicLib.nsh"

; Icons
!define MUI_ICON "icon.ico"
!define MUI_UNICON "icon.ico"

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

; Pre-installation checks
Function .onInit
    ; Check if already installed
    ReadRegStr $0 HKCU "Software\Rewind" ""
    ${If} $0 != ""
        ; Found installed version
        MessageBox MB_YESNO|MB_ICONQUESTION "Rewind is already installed. Do you want to uninstall the current version and reinstall?" IDYES uninstall IDNO done

        uninstall:
            ; Close running application
            DetailPrint "Checking for running Rewind application..."
            nsExec::Exec 'taskkill /F /IM Rewind.exe /T'
            Sleep 1000

            ; Run old uninstaller
            ${If} ${FileExists} "$0\uninstall.exe"
                DetailPrint "Uninstalling previous version..."
                ExecWait '"$0\uninstall.exe" /S _?=$0'
                Sleep 500
            ${EndIf}
            Goto done

        done:
    ${EndIf}
FunctionEnd

Section "Rewind" SecRewind
    ; Check and close running application again
    nsExec::Exec 'taskkill /F /IM Rewind.exe /T'
    Sleep 500

    SetOutPath "$INSTDIR"
    
    ; Main executable
    File "Rewind.exe"

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
    CreateShortcut "$DESKTOP\Rewind.lnk" "$INSTDIR\Rewind.exe"
SectionEnd

Section "Uninstall"
    Delete "$INSTDIR\Rewind.exe"
    Delete "$INSTDIR\uninstall.exe"
    Delete "$INSTDIR\bin\ffmpeg.exe"
    RMDir "$INSTDIR\bin"
    RMDir "$INSTDIR"
    Delete "$DESKTOP\Rewind.lnk"
    DeleteRegKey HKCU "Software\Rewind"
    DeleteRegKey HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\Rewind"
SectionEnd
