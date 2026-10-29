@echo off
setlocal
cd /d "%~dp0"
Lime.exe %*
if errorlevel 1 pause
