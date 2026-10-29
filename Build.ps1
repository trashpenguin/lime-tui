param([string]$GoExecutable = 'go', [string]$Version)
$ErrorActionPreference = 'Stop'
$root = $PSScriptRoot
if (!$Version) {$Version=(Get-Content -LiteralPath (Join-Path $root 'VERSION') -Raw).Trim()}
function Invoke-Checked {
    param([string]$Tool, [string[]]$ToolArgs)
    & $Tool @ToolArgs
    if ($LASTEXITCODE -ne 0) { throw "$Tool failed ($LASTEXITCODE)" }
}
if ($Version -notmatch '^\d+\.\d+\.\d+([-.][A-Za-z0-9.]+)?$') { throw 'Use a semantic version.' }
$release = Join-Path $root "dist\LimeTUI-$Version-windows-x64"
New-Item -ItemType Directory -Force -Path $release | Out-Null
Invoke-Checked -Tool cargo -ToolArgs @('build','--locked','--release','--manifest-path',(Join-Path $root 'backend\Cargo.toml'),'--no-default-features','--features','pomme-bridge','--bin','spotify-engine','-j','4')
& (Join-Path $root 'scripts/Build-Resources.ps1') -GoExecutable $GoExecutable -Version $Version
Invoke-Checked -Tool $GoExecutable -ToolArgs @('-C',(Join-Path $root 'frontend'),'build','-trimpath','-ldflags',"-s -w -X main.version=$Version",'-o',(Join-Path $release 'Lime.exe'),'.')
Copy-Item -LiteralPath (Join-Path $root 'backend\target\release\spotify-engine.exe') -Destination (Join-Path $release 'lime-engine.exe')
foreach ($file in @('VERSION','LICENSE','CHANGELOG.md','README.md','NOTICE.md','LICENSE-FRONTEND.txt','LICENSE-SPOTATUI.txt','THIRD-PARTY-NOTICES.txt','VALIDATION.md','SPOTATUI-FEATURES.md','RELEASE-READINESS.md','PRIVACY.md','Start Lime.cmd','Create Lime Shortcut.ps1','SIGNING.md','Sign-Windows.ps1','SPOTIFY-DISTRIBUTION-REVIEW.md','DEPENDENCY-ASSESSMENTS.json','VEX.json','DEPENDENCY-AUDIT.json')) {
    if (Test-Path -LiteralPath (Join-Path $root $file)) { Copy-Item -LiteralPath (Join-Path $root $file) -Destination $release }
}
Copy-Item -LiteralPath (Join-Path $root 'assets') -Destination $release -Recurse -Force
Get-ChildItem -LiteralPath $release -Filter '*.exe' | ForEach-Object { '{0}  {1}' -f (Get-FileHash -LiteralPath $_.FullName -Algorithm SHA256).Hash.ToLowerInvariant(), $_.Name } | Set-Content -LiteralPath (Join-Path $release 'SHA256SUMS.txt') -Encoding ascii
Write-Host "Portable app: $release"
