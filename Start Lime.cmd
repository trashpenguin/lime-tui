@echo off
setlocal
cd /d "%~dp0"
if "%~1"=="" (
  Lime.exe --window
) else (
  Lime.exe %*
)
if errorlevel 1 pause
