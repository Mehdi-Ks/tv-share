; TV Share installer (NSIS). Per-user install: no admin prompt.
; Build: makensis /DVERSION=1.0.0 build\installer\tvshare.nsi   (after `wails build`)

Unicode true
SetCompressor /SOLID lzma

!ifndef VERSION
  !define VERSION "1.8.1"
!endif
!define APPNAME "TV Share"
!define EXE "TVShare.exe"
!define UNINST_KEY "Software\Microsoft\Windows\CurrentVersion\Uninstall\TVShare"
!define WEBVIEW2_GUID "{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}"

Name "${APPNAME}"
OutFile "..\bin\TVShare-Setup.exe"
InstallDir "$LOCALAPPDATA\Programs\${APPNAME}"
InstallDirRegKey HKCU "${UNINST_KEY}" "InstallLocation"
RequestExecutionLevel user
ShowInstDetails nevershow
ShowUninstDetails nevershow
BrandingText "${APPNAME} ${VERSION}"

VIProductVersion "${VERSION}.0"
VIAddVersionKey "ProductName" "${APPNAME}"
VIAddVersionKey "ProductVersion" "${VERSION}"
VIAddVersionKey "FileVersion" "${VERSION}"
VIAddVersionKey "FileDescription" "${APPNAME} installer"
VIAddVersionKey "LegalCopyright" "MIT License"

!include "MUI2.nsh"
!include "LogicLib.nsh"

!define MUI_ICON "..\windows\icon.ico"
!define MUI_UNICON "..\windows\icon.ico"
!define MUI_ABORTWARNING
!define MUI_WELCOMEFINISHPAGE_BITMAP "welcome.bmp"
!define MUI_UNWELCOMEFINISHPAGE_BITMAP "welcome.bmp"
!define MUI_FINISHPAGE_RUN "$INSTDIR\${EXE}"
!define MUI_FINISHPAGE_RUN_TEXT "$(RunNow)"

!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH
!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES

!insertmacro MUI_LANGUAGE "English"
!insertmacro MUI_LANGUAGE "French"

LangString RunNow ${LANG_ENGLISH} "Open TV Share now"
LangString RunNow ${LANG_FRENCH} "Ouvrir TV Share maintenant"
LangString NeedWebView ${LANG_ENGLISH} "TV Share needs Microsoft Edge WebView2, which is missing on this PC.$\r$\n$\r$\nClick OK to open Microsoft's download page. Install it, then run this setup again."
LangString NeedWebView ${LANG_FRENCH} "TV Share a besoin de Microsoft Edge WebView2, absent de ce PC.$\r$\n$\r$\nCliquez sur OK pour ouvrir la page de téléchargement de Microsoft. Installez-le, puis relancez cette installation."

; NSIS picks English or French automatically from the Windows display language.

; WebView2 ships with Windows 11 and almost every up-to-date Windows 10.
Function CheckWebView2
  ReadRegStr $0 HKLM "SOFTWARE\WOW6432Node\Microsoft\EdgeUpdate\Clients\${WEBVIEW2_GUID}" "pv"
  ${If} $0 == ""
  ${OrIf} $0 == "0.0.0.0"
    ReadRegStr $0 HKCU "Software\Microsoft\EdgeUpdate\Clients\${WEBVIEW2_GUID}" "pv"
  ${EndIf}
  ${If} $0 == ""
  ${OrIf} $0 == "0.0.0.0"
    MessageBox MB_OKCANCEL|MB_ICONEXCLAMATION "$(NeedWebView)" IDCANCEL +2
      ExecShell "open" "https://developer.microsoft.com/microsoft-edge/webview2/#download"
    Abort
  ${EndIf}
FunctionEnd

Section "Install"
  Call CheckWebView2

  ; Close a running copy so its file can be replaced (this also stops sharing)
  nsExec::Exec 'taskkill /im ${EXE} /f'
  Sleep 500

  SetOutPath "$INSTDIR"
  File "..\bin\${EXE}"
  WriteUninstaller "$INSTDIR\uninstall.exe"

  SetShellVarContext current
  CreateShortCut "$SMPROGRAMS\${APPNAME}.lnk" "$INSTDIR\${EXE}"
  CreateShortCut "$DESKTOP\${APPNAME}.lnk" "$INSTDIR\${EXE}"

  ; "Installed apps" entry
  WriteRegStr HKCU "${UNINST_KEY}" "DisplayName" "${APPNAME}"
  WriteRegStr HKCU "${UNINST_KEY}" "DisplayVersion" "${VERSION}"
  WriteRegStr HKCU "${UNINST_KEY}" "Publisher" "TV Share (open source)"
  WriteRegStr HKCU "${UNINST_KEY}" "URLInfoAbout" "https://github.com/Mehdi-Ks/tv-share"
  WriteRegStr HKCU "${UNINST_KEY}" "DisplayIcon" "$INSTDIR\${EXE}"
  WriteRegStr HKCU "${UNINST_KEY}" "InstallLocation" "$INSTDIR"
  WriteRegStr HKCU "${UNINST_KEY}" "UninstallString" '"$INSTDIR\uninstall.exe"'
  WriteRegStr HKCU "${UNINST_KEY}" "QuietUninstallString" '"$INSTDIR\uninstall.exe" /S'
  WriteRegDWORD HKCU "${UNINST_KEY}" "NoModify" 1
  WriteRegDWORD HKCU "${UNINST_KEY}" "NoRepair" 1
  WriteRegDWORD HKCU "${UNINST_KEY}" "EstimatedSize" 12000
SectionEnd

Section "Uninstall"
  nsExec::Exec 'taskkill /im ${EXE} /f'
  Sleep 500

  SetShellVarContext current
  Delete "$SMPROGRAMS\${APPNAME}.lnk"
  Delete "$DESKTOP\${APPNAME}.lnk"
  Delete "$INSTDIR\${EXE}"
  Delete "$INSTDIR\uninstall.exe"
  RMDir "$INSTDIR"

  ; "Open with Windows" setting, if it was turned on
  DeleteRegValue HKCU "Software\Microsoft\Windows\CurrentVersion\Run" "${APPNAME}"
  DeleteRegKey HKCU "${UNINST_KEY}"
  ; Settings in %APPDATA%\TV Share are kept, like most apps, in case of a reinstall.
  ; rclone and FFmpeg stay installed: other programs may use them.
SectionEnd
