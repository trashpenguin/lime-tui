# Windows publisher signing

This preview is unsigned. No usable code-signing certificate was found in the current-user or local-machine personal stores, and the publisher confirmed they do not have one. A self-signed development certificate would not provide public publisher trust; none was created or installed.

Obtain a publicly trusted code-signing identity from a certificate authority or a supported managed signing provider. Enrollment requires your real publisher identity and may involve verification and charges. Do not send private keys or passwords in chat. Keep hardware-token/PIN steps under your control. A managed signing service may require its own signing client instead of the local-certificate script below.

For a certificate available through the Windows certificate store, build the final release and run from the matching source folder:

```powershell
.\Sign-Windows.ps1 -ReleaseDirectory C:\path\to\LimeTUI-1.0.0-windows-x64 -Thumbprint YOUR_CERTIFICATE_THUMBPRINT
```

The script requires code-signing usage, a valid certificate with private-key access, SHA-256 file and RFC3161 timestamp digests, a trusted timestamp and successful Windows signature verification for both EXEs. It then recomputes binary checksums. It never creates a certificate or changes Windows trust settings. Your signing provider may require a different timestamp server; use `-TimestampServer` as needed. The default HTTPS RFC3161 endpoint is provided by DigiCert's timestamp service. Microsoft Windows SDK SignTool is required; it is discovered locally or supplied with -SignTool.

Inspect signatures without signing:

```powershell
.\Sign-Windows.ps1 -ReleaseDirectory C:\path\to\release -CheckOnly
```

After signing, re-create the portable ZIP and verify its bytes/checksums. Signing changes binary hashes. Re-run antivirus/release checks against those exact signed files. Signing identifies a publisher; it does not guarantee immediate SmartScreen reputation, certify Spotify permission, or replace security testing. Do not label an unsigned or invalidly signed build as signed.
