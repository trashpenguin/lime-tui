param([switch]$Soloist)

$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'
# Use Windows PowerShell's inbox modules even when setup starts from PowerShell 7.
$env:PSModulePath = Join-Path $PSHOME 'Modules'
$runtimeKey = 'HKLM:\SOFTWARE\Microsoft\VisualStudio\14.0\VC\Runtimes\x64'

function Test-VCRuntime {
    $runtime = Get-ItemProperty -LiteralPath $runtimeKey -ErrorAction SilentlyContinue
    return $runtime.Installed -eq 1
}

function Assert-TrustedPublisher([string]$Path, [string]$Publisher) {
    $signature = Get-AuthenticodeSignature -LiteralPath $Path
    if ($signature.Status -ne 'Valid' -or $null -eq $signature.SignerCertificate -or
        $signature.SignerCertificate.GetNameInfo([Security.Cryptography.X509Certificates.X509NameType]::SimpleName, $false) -ne $Publisher) {
        throw "The download failed $Publisher publisher verification."
    }
}

function Install-Soloist {
    $archivePath = Join-Path $PSScriptRoot 'soloist-windows-x64.zip'
    $binary = Join-Path $PSScriptRoot 'soloist.exe'
    $licenses = Join-Path $PSScriptRoot 'soloist-licenses.txt'
    Invoke-WebRequest -Uri 'https://soloist-builds.spotifycdn.com/soloist_release_windows_x86_64.zip' -OutFile $archivePath -UseBasicParsing -TimeoutSec 300

    # Extract only the expected files to fixed names inside setup's temporary directory.
    Add-Type -AssemblyName System.IO.Compression.FileSystem
    $archive = [IO.Compression.ZipFile]::OpenRead($archivePath)
    try {
        foreach ($file in @{ 'soloist.exe' = $binary; 'THIRD_PARTY_LICENSES.txt' = $licenses }.GetEnumerator()) {
            $entry = $archive.GetEntry($file.Key)
            if (!$entry -or $entry.Length -eq 0) { throw "The Spotify download is missing $($file.Key)." }
            [IO.Compression.ZipFileExtensions]::ExtractToFile($entry, $file.Value, $true)
        }
    } finally {
        $archive.Dispose()
    }
    Assert-TrustedPublisher -Path $binary -Publisher 'Spotify AB'
}

try {
    [Net.ServicePointManager]::SecurityProtocol = [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12
    if ($Soloist) {
        Install-Soloist
        exit 0
    }
    if (Test-VCRuntime) { exit 0 }

    $installer = Join-Path $PSScriptRoot 'vc_redist.x64.exe'
    Invoke-WebRequest -Uri 'https://aka.ms/vc14/vc_redist.x64.exe' -OutFile $installer -UseBasicParsing -TimeoutSec 300
    Assert-TrustedPublisher -Path $installer -Publisher 'Microsoft Corporation'
    $process = Start-Process -FilePath $installer -ArgumentList '/install', '/quiet', '/norestart' -Verb RunAs -PassThru -WindowStyle Hidden
    try {
        if (!$process.WaitForExit(900000)) { throw 'The Microsoft runtime installer is still running. Wait for it to finish and retry setup.' }
        $code = $process.ExitCode
    } finally {
        $process.Dispose()
    }

    if ($code -eq 3010) { exit 3010 }
    if ($code -ne 0) { throw "The Microsoft runtime installer returned exit code $code." }
    if (!(Test-VCRuntime)) { throw 'Windows did not register the Microsoft Visual C++ x64 Runtime.' }
    exit 0
} catch {
    $message = $_.Exception.Message
    $message | Set-Content -LiteralPath (Join-Path $PSScriptRoot 'dependency-error.txt') -Encoding ascii
    exit 1
}
