#ifndef AppVersion
  #error AppVersion must be provided by Build-Setup.ps1
#endif
#ifndef ReleaseDir
  #error ReleaseDir must be provided by Build-Setup.ps1
#endif
#ifndef OutputDir
  #error OutputDir must be provided by Build-Setup.ps1
#endif

[Setup]
AppId={{D44B7684-A522-4965-A061-8C0B96866B69}
AppName=Lime TUI
AppVersion={#AppVersion}
AppPublisher=Lime TUI contributors
AppPublisherURL=https://github.com/trashpenguin/lime-tui
AppSupportURL=https://github.com/trashpenguin/lime-tui/issues
AppUpdatesURL=https://github.com/trashpenguin/lime-tui/releases
DefaultDirName={localappdata}\Programs\LimeTUI
DefaultGroupName=Lime TUI
PrivilegesRequired=lowest
SetupArchitecture=x64
ArchitecturesAllowed=x64os
ArchitecturesInstallIn64BitMode=x64os
MinVersion=10.0
DisableProgramGroupPage=yes
LicenseFile=..\LICENSE
SetupIconFile=..\assets\lime.ico
UninstallDisplayIcon={app}\Lime.exe
OutputDir={#OutputDir}
OutputBaseFilename=LimeTUI-{#AppVersion}-setup-x64
Compression=lzma2
SolidCompression=yes
WizardStyle=modern
CloseApplications=yes
RestartApplications=no
SetupMutex=LimeTUISetup

[Files]
; Explicit payload prevents local profiles and reports from entering setup.
Source: "{#ReleaseDir}\Lime.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#ReleaseDir}\lime-engine.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#ReleaseDir}\Start Lime.cmd"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#ReleaseDir}\Setup Soloist.cmd"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#ReleaseDir}\VERSION"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#ReleaseDir}\SHA256SUMS.txt"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#ReleaseDir}\LICENSE"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#ReleaseDir}\LICENSE-FRONTEND.txt"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#ReleaseDir}\LICENSE-SPOTATUI.txt"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#ReleaseDir}\NOTICE.md"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#ReleaseDir}\THIRD-PARTY-NOTICES.txt"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#ReleaseDir}\README.md"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#ReleaseDir}\SOLOIST-SETUP.md"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#ReleaseDir}\CHANGELOG.md"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#ReleaseDir}\PRIVACY.md"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#ReleaseDir}\RELEASE-READINESS.md"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#ReleaseDir}\SPOTIFY-DISTRIBUTION-REVIEW.md"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#ReleaseDir}\VALIDATION.md"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#ReleaseDir}\assets\lime.png"; DestDir: "{app}\assets"; Flags: ignoreversion
Source: "{tmp}\soloist.exe"; DestDir: "{app}\Spotify Soloist"; Flags: external ignoreversion; Check: InstallSoloist
Source: "{tmp}\soloist-licenses.txt"; DestDir: "{app}\Spotify Soloist"; DestName: "THIRD_PARTY_LICENSES.txt"; Flags: external ignoreversion; Check: InstallSoloist
Source: "Install-Dependencies.ps1"; Flags: dontcopy

[Icons]
Name: "{group}\Lime TUI"; Filename: "{app}\Lime.exe"; Parameters: "--window"; WorkingDir: "{app}"
Name: "{group}\Configure Spotify Soloist"; Filename: "{app}\Setup Soloist.cmd"; WorkingDir: "{app}"; IconFilename: "{app}\Lime.exe"

[Run]
Filename: "{app}\Lime.exe"; Parameters: "--window"; Description: "Open Lime and connect to Spotify"; Flags: postinstall nowait skipifsilent; Check: LaunchLime
Filename: "{app}\Setup Soloist.cmd"; Description: "Open Lime's Soloist configuration"; Flags: shellexec postinstall nowait skipifsilent; Check: LaunchSoloistSetup

[Code]
var
  PlaybackPage: TInputOptionWizardPage;
  RestartRequired: Boolean;

procedure OpenSpotifyPage(Url: String);
var
  Code: Integer;
begin
  if not ShellExec('open', Url, '', '', SW_SHOWNORMAL, ewNoWait, Code) then
    MsgBox('Open this address in your browser:' + #13#10 + Url, mbInformation, MB_OK);
end;

procedure OpenDownloads(Sender: TObject);
begin
  OpenSpotifyPage('https://developer.spotify.com/documentation/soloist/reference/downloads-and-updates');
end;

procedure OpenKeyDashboard(Sender: TObject);
begin
  OpenSpotifyPage('https://developer.spotify.com/dashboard/soloist');
end;

procedure AddPlaybackButton(Caption: String; Offset: Integer; OnClick: TNotifyEvent);
var
  Button: TNewButton;
begin
  Button := TNewButton.Create(PlaybackPage.Surface);
  Button.Parent := PlaybackPage.Surface;
  Button.Caption := Caption;
  Button.Left := ScaleX(Offset);
  Button.Top := PlaybackPage.SurfaceHeight - ScaleY(30);
  Button.Width := ScaleX(170);
  Button.Height := ScaleY(25);
  Button.OnClick := OnClick;
end;

procedure InitializeWizard;
begin
  PlaybackPage := CreateInputOptionPage(wpLicense, 'Spotify playback', 'Choose what to open after installation',
    'Lime needs Spotify Premium and internet access. Setup installs the Microsoft Visual C++ x64 Runtime if missing. ' +
    'Windows may ask for administrator permission.' + #13#10#13#10 +
    'If native playback gives an audio-key error, choose Soloist. Setup downloads the official Windows build directly from Spotify. ' +
    'Generate your own key, then pair Lime Soloist in the Spotify app on the same network.',
    True, False);
  PlaybackPage.Add('Open Lime');
  PlaybackPage.Add('Install and configure Spotify Soloist');
  if CompareText(ExpandConstant('{param:PLAYBACK|native}'), 'soloist') = 0 then
    PlaybackPage.SelectedValueIndex := 1
  else
    PlaybackPage.SelectedValueIndex := 0;
  AddPlaybackButton('Spotify Soloist download', 0, @OpenDownloads);
  AddPlaybackButton('Get your Soloist key', 178, @OpenKeyDashboard);
  ExtractTemporaryFile('Install-Dependencies.ps1');
end;

function InstallSoloist: Boolean;
begin
  Result := PlaybackPage.SelectedValueIndex = 1;
end;

function RunDependencySetup(Arguments, Name: String; var Code: Integer): String;
var
  Details: AnsiString;
begin
  Result := '';
  DeleteFile(ExpandConstant('{tmp}\dependency-error.txt'));
  if not Exec(ExpandConstant('{sys}\WindowsPowerShell\v1.0\powershell.exe'),
    '-NoProfile -NonInteractive -ExecutionPolicy Bypass -File "' +
    ExpandConstant('{tmp}\Install-Dependencies.ps1') + '"' + Arguments,
    '', SW_HIDE, ewWaitUntilTerminated, Code) then
    Result := 'Could not start ' + Name + ' setup. ' + SysErrorMessage(Code)
  else if (Code <> 0) and ((Code <> 3010) or (Arguments <> '')) then begin
    Result := 'Could not install ' + Name + '.';
    if LoadStringFromFile(ExpandConstant('{tmp}\dependency-error.txt'), Details) then
      Result := Result + #13#10 + String(Details);
    Result := Result + #13#10 + 'Check your connection, allow any Windows permission prompts, and retry.';
  end;
end;

function PrepareToInstall(var NeedsRestart: Boolean): String;
var
  Code: Integer;
begin
  WizardForm.PreparingLabel.Caption := 'Checking the Microsoft Visual C++ x64 Runtime. Allow the Windows permission prompt if shown.';
  Result := RunDependencySetup('', 'Microsoft Visual C++ x64 Runtime', Code);
  if Result <> '' then Exit;
  if Code = 3010 then RestartRequired := True;

  if InstallSoloist then begin
    WizardForm.PreparingLabel.Caption := 'Downloading Spotify Soloist and verifying its publisher.';
    Result := RunDependencySetup(' -Soloist', 'Spotify Soloist', Code);
  end;
end;

function LaunchLime: Boolean;
begin
  Result := not RestartRequired and (PlaybackPage.SelectedValueIndex = 0);
end;

function LaunchSoloistSetup: Boolean;
begin
  Result := not RestartRequired and (PlaybackPage.SelectedValueIndex = 1);
end;

procedure CurPageChanged(CurPageID: Integer);
begin
  if (CurPageID = wpFinished) and RestartRequired then
    WizardForm.FinishedLabel.Caption :=
      'Lime is installed. Restart Windows to finish installing the Microsoft runtime, then open Lime or Configure Spotify Soloist from Start.';
end;

function GetCustomSetupExitCode: Integer;
begin
  if RestartRequired then Result := 3010 else Result := 0;
end;
