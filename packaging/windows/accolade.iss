; Inno Setup script for Accolade. Built by .github/workflows/release.yml:
;   iscc /DAppVersion=0.2.0 /DSourceDir=<dir with Accolade.exe> packaging\windows\accolade.iss

#ifndef AppVersion
  #define AppVersion "0.0.0"
#endif
#ifndef SourceDir
  #define SourceDir "..\.."
#endif

[Setup]
AppId={{6A3B7F0E-4C1D-4B5E-9A1F-ACC01ADE0001}
AppName=Accolade
AppVersion={#AppVersion}
AppPublisher=Joop Kiefte
AppPublisherURL=https://github.com/LaPingvino/accolade
DefaultDirName={autopf}\Accolade
DefaultGroupName=Accolade
UninstallDisplayIcon={app}\Accolade.exe
OutputDir=..\..\dist
OutputBaseFilename=Accolade-{#AppVersion}-Setup
Compression=lzma2
SolidCompression=yes
WizardStyle=modern
; install for the current user unless the user chooses all users
PrivilegesRequired=lowest
PrivilegesRequiredOverridesAllowed=dialog
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
LicenseFile=..\..\LICENSE

[Tasks]
Name: "desktopicon"; Description: "{cm:CreateDesktopIcon}"; GroupDescription: "{cm:AdditionalIcons}"; Flags: unchecked
Name: "fountainassoc"; Description: "Open .fountain files with Accolade"; GroupDescription: "File associations:"

[Files]
Source: "{#SourceDir}\Accolade.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\..\LICENSE"; DestDir: "{app}"; DestName: "LICENSE.txt"; Flags: ignoreversion

[Icons]
Name: "{group}\Accolade"; Filename: "{app}\Accolade.exe"
Name: "{group}\{cm:UninstallProgram,Accolade}"; Filename: "{uninstallexe}"
Name: "{autodesktop}\Accolade"; Filename: "{app}\Accolade.exe"; Tasks: desktopicon

[Registry]
Root: HKA; Subkey: "Software\Classes\.fountain\OpenWithProgids"; ValueType: string; ValueName: "Accolade.Fountain"; ValueData: ""; Flags: uninsdeletevalue; Tasks: fountainassoc
Root: HKA; Subkey: "Software\Classes\Accolade.Fountain"; ValueType: string; ValueName: ""; ValueData: "Fountain screenplay"; Flags: uninsdeletekey; Tasks: fountainassoc
Root: HKA; Subkey: "Software\Classes\Accolade.Fountain\DefaultIcon"; ValueType: string; ValueName: ""; ValueData: "{app}\Accolade.exe,0"; Tasks: fountainassoc
Root: HKA; Subkey: "Software\Classes\Accolade.Fountain\shell\open\command"; ValueType: string; ValueName: ""; ValueData: """{app}\Accolade.exe"" ""%1"""; Tasks: fountainassoc

[Run]
Filename: "{app}\Accolade.exe"; Description: "{cm:LaunchProgram,Accolade}"; Flags: nowait postinstall skipifsilent
