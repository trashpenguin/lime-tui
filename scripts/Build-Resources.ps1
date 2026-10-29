param([string]$GoExecutable='go', [string]$Version)
$ErrorActionPreference='Stop'
$root=Split-Path $PSScriptRoot -Parent
if (!$Version) {$Version=(Get-Content -LiteralPath (Join-Path $root 'VERSION') -Raw).Trim()}
& $GoExecutable run github.com/tc-hib/go-winres@v0.3.3 make --in (Join-Path $root 'assets/winres.json') --arch amd64 --out (Join-Path $root 'frontend/lime') --file-version $Version --product-version $Version
if ($LASTEXITCODE -ne 0) {throw 'Windows resource generation failed'}
