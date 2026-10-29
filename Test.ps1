param([string]$GoExecutable = 'go', [switch]$Security, [string]$PythonExecutable = 'python')
$ErrorActionPreference = 'Stop'
if ($env:NATIVE_LIVE_TEST -eq '1') {throw 'The offline release gate refuses live-audio opt-in. Run live tests separately with an isolated state directory.'}
& $GoExecutable -C (Join-Path $PSScriptRoot 'frontend') test ./...
if ($LASTEXITCODE -ne 0) { throw 'Frontend tests failed.' }
& cargo test --locked --manifest-path (Join-Path $PSScriptRoot 'backend\Cargo.toml') --no-default-features --features pomme-bridge --lib -j 4
if ($LASTEXITCODE -ne 0) { throw 'Backend tests failed.' }
& cargo clippy --locked --manifest-path (Join-Path $PSScriptRoot 'backend\Cargo.toml') --no-default-features --features pomme-bridge --bin spotify-engine -- -D warnings
if ($LASTEXITCODE -ne 0) { throw 'Backend lint failed.' }

& $GoExecutable -C (Join-Path $PSScriptRoot 'frontend') vet ./...
if ($LASTEXITCODE -ne 0) {throw 'Frontend vet failed.'}
if ($Security) {
 & $GoExecutable -C (Join-Path $PSScriptRoot 'frontend') run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...
 if ($LASTEXITCODE -ne 0) {throw 'Go vulnerability scan failed.'}
 & $PythonExecutable (Join-Path $PSScriptRoot 'scripts/Audit-Dependencies.py') --out (Join-Path $PSScriptRoot 'DEPENDENCY-AUDIT.json')
 if ($LASTEXITCODE -ne 0) {throw 'Unassessed Rust advisories or dependency audit failure.'}
}
