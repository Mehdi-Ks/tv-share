# Builds TVShare.exe and TVShare-Setup.exe for a release.
#
# Official posters need a TMDB key. Put it in build\tmdb.key (one line; that file is never
# committed, see .gitignore). Without it the app still builds: people can paste their own key
# in Settings, and posters fall back to frames from the movies.
#
#   powershell -ExecutionPolicy Bypass -File build.ps1
param([string]$Version = "")
$ErrorActionPreference = "Stop"
Set-Location $PSScriptRoot

if (-not $Version) { $Version = (Select-String -Path app.go -Pattern 'appVersion = "(.+)"').Matches[0].Groups[1].Value }

$ldflags = ""
$keyFile = Join-Path $PSScriptRoot "build\tmdb.key"
if (Test-Path $keyFile) {
  $key = (Get-Content $keyFile -Raw).Trim()
  if ($key -notmatch '^[A-Za-z0-9._-]+$') { throw "build\tmdb.key doesn't look like a TMDB key." }
  $ldflags = "-X main.tmdbBuiltInKey=$key"
  Write-Host "TMDB key: found (built in)."
} else {
  Write-Host "TMDB key: none (build\tmdb.key missing). Official posters need a key pasted in Settings."
}

go test ./...
if ($LASTEXITCODE) { throw "tests failed" }

$wails = Join-Path (go env GOPATH) "bin\wails.exe"
# Wails prints its build settings, key included: hide it so it never lands in a log.
if ($ldflags) { & $wails build -clean -trimpath -ldflags $ldflags 2>&1 | ForEach-Object { "$_".Replace($key, "********") } }
else { & $wails build -clean -trimpath }
if ($LASTEXITCODE) { throw "wails build failed" }

$nsis = (Get-Command makensis -ErrorAction SilentlyContinue).Source
if (-not $nsis) { $nsis = "${env:ProgramFiles(x86)}\NSIS\makensis.exe" }
& $nsis /V2 "/DVERSION=$Version" build\installer\tvshare.nsi
if ($LASTEXITCODE) { throw "installer build failed" }

Write-Host "Built TV Share $Version in build\bin."
