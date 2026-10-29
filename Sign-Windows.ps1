param(
    [Parameter(Mandatory=$true)][string]$ReleaseDirectory,
    [string]$Thumbprint,
    [ValidateSet('CurrentUser','LocalMachine')][string]$StoreLocation='CurrentUser',
    [string]$TimestampServer='https://timestamp.digicert.com',
    [string]$SignTool,
    [switch]$CheckOnly
)
$ErrorActionPreference='Stop'
$directory=(Resolve-Path -LiteralPath $ReleaseDirectory).Path
$files=@('Lime.exe','lime-engine.exe') | ForEach-Object {
    $file=Join-Path $directory $_
    if (!(Test-Path -LiteralPath $file -PathType Leaf)) {throw "Missing release executable: $_"}
    $file
}
if ($CheckOnly) {
    $files | ForEach-Object {Get-AuthenticodeSignature -LiteralPath $_} | Select-Object Path,Status,StatusMessage
    return
}
if ($Thumbprint -notmatch '^[A-Fa-f0-9]{40}$') {throw 'Supply the Windows thumbprint of your installed publisher code-signing certificate.'}
$certificate=Get-Item -LiteralPath "Cert:\$StoreLocation\My\$Thumbprint"
if (!$certificate.HasPrivateKey) {throw 'The selected certificate has no accessible private key.'}
if ($certificate.NotBefore -gt (Get-Date) -or $certificate.NotAfter -lt (Get-Date)) {throw 'The signing certificate is not currently valid.'}
if (!($certificate.EnhancedKeyUsageList | Where-Object {$_.ObjectId -eq '1.3.6.1.5.5.7.3.3'})) {throw 'The certificate must explicitly allow code signing.'}
if (!$SignTool) {
    $sdk=Get-ChildItem "${env:ProgramFiles(x86)}/Windows Kits/10/bin/*/x64/signtool.exe" -ErrorAction SilentlyContinue | Sort-Object FullName | Select-Object -Last 1
    if (!$sdk) {throw 'Install the Windows SDK signing tools or pass -SignTool with the full executable path.'}
    $SignTool=$sdk.FullName
}
if (!(Test-Path -LiteralPath $SignTool -PathType Leaf)) {throw 'SignTool executable was not found.'}
if (!([uri]$TimestampServer).IsAbsoluteUri -or ([uri]$TimestampServer).Scheme -ne 'https') {throw 'Use an HTTPS RFC3161 timestamp server.'}
foreach ($file in $files) {
    $arguments=@('sign','/fd','SHA256','/tr',$TimestampServer,'/td','SHA256','/s','My','/sha1',$Thumbprint)
    if ($StoreLocation -eq 'LocalMachine') {$arguments+='/sm'}
    $arguments+=$file
    & $SignTool @arguments
    if ($LASTEXITCODE -ne 0) {throw "Signing failed for $file. Do not publish this folder."}
    & $SignTool verify /pa /all /v $file
    if ($LASTEXITCODE -ne 0) {throw "Signature verification failed for $file. Do not publish this folder."}
    $signed=Get-AuthenticodeSignature -LiteralPath $file
    if ($signed.Status -ne 'Valid' -or !$signed.TimeStamperCertificate -or $signed.SignerCertificate.Thumbprint -ne $Thumbprint) {throw "Publisher/timestamp verification failed for $file. Do not publish this folder."}
}
$files | ForEach-Object {
    $signature=Get-AuthenticodeSignature -LiteralPath $_
    if ($signature.Status -ne 'Valid') {throw "Final signature verification failed: $_"}
    '{0}  {1}' -f (Get-FileHash -LiteralPath $_ -Algorithm SHA256).Hash.ToLowerInvariant(),(Split-Path $_ -Leaf)
} | Set-Content -LiteralPath (Join-Path $directory 'SHA256SUMS.txt') -Encoding ascii
Write-Host 'Both EXEs signed, timestamped and verified. Checksums updated. Re-create and verify the ZIP before publishing.'
