param(
    [string]$Iscc = 'ISCC.exe',
    [string]$Version,
    [string]$ReleaseDirectory
)

$ErrorActionPreference = 'Stop'
$root = Split-Path $PSScriptRoot -Parent
if (!$Version) { $Version = (Get-Content -LiteralPath (Join-Path $root 'VERSION') -Raw).Trim() }
if ($Version -notmatch '^\d+\.\d+\.\d+([-.][A-Za-z0-9.]+)?$') { throw 'Use a semantic version.' }

$output = Join-Path $root 'dist'
if (!$ReleaseDirectory) { $ReleaseDirectory = Join-Path $output "LimeTUI-$Version-windows-x64" }
$release = (Resolve-Path -LiteralPath $ReleaseDirectory).Path
foreach ($file in @('Lime.exe', 'lime-engine.exe', 'Setup Soloist.cmd', 'LICENSE', 'THIRD-PARTY-NOTICES.txt', 'VERSION')) {
    if (!(Test-Path -LiteralPath (Join-Path $release $file) -PathType Leaf)) {
        throw "Missing $file in the portable build. Run Build.ps1 first."
    }
}
if ((Get-Content -LiteralPath (Join-Path $release 'VERSION') -Raw).Trim() -ne $Version) {
    throw 'The portable build version does not match the setup version.'
}

& $Iscc "/DAppVersion=$Version" "/DReleaseDir=$release" "/DOutputDir=$output" (Join-Path $PSScriptRoot 'Lime.Setup.iss')
if ($LASTEXITCODE -ne 0) { throw "Inno Setup failed ($LASTEXITCODE)." }
Write-Host "Windows setup: $(Join-Path $output "LimeTUI-$Version-setup-x64.exe")"
