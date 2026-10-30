param([string]$Destination = (Join-Path $env:APPDATA 'Microsoft/Windows/Start Menu/Programs/Lime TUI.lnk'))
$ErrorActionPreference = 'Stop'
$exe=Join-Path $PSScriptRoot 'Lime.exe'
if (!(Test-Path -LiteralPath $exe -PathType Leaf)) {throw 'Extract the complete Lime ZIP first, then run this script beside Lime.exe.'}
$folder=Split-Path $Destination -Parent
if (!(Test-Path -LiteralPath $folder)) {throw 'Shortcut destination directory does not exist.'}
$shell=New-Object -ComObject WScript.Shell
$shortcut=$shell.CreateShortcut($Destination)
$shortcut.TargetPath=$exe
$shortcut.Arguments='--window'
$shortcut.WorkingDirectory=$PSScriptRoot
$shortcut.IconLocation="$exe,0"
$shortcut.Description='Lime TUI - Your music. Your terminal.'
$shortcut.Save()
Write-Host "Created $Destination"
