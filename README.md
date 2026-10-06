# Port Monitor

A small Windows desktop utility built with Go + Wails.

It detects all currently listening TCP ports, shows the owning PID/process, and lets the user terminate a selected process after confirmation.

## Features

- Scans all TCP `LISTENING` ports using Windows `netstat`.
- Resolves PID to process name using Windows `tasklist`.
- Kill button for every detected listener.
- Confirmation before termination.
- Uses `taskkill /PID /F` for Windows-native process termination.
- Detects access-denied/elevation failures.
- Can relaunch itself through normal Windows UAC using the `runas` elevation mechanism.
- Never asks for, collects, or stores a Windows password.
- Refresh button.
- Optional 5-second auto-refresh.
- Search/filter by port, PID, or process name.
- Handles ports/processes disappearing between scan and kill.
- No hard-coded port list.

## Requirements for building

Build on Windows.

Install:

1. Go 1.22 or newer
2. Node.js 18+ / npm
3. Wails v2 CLI
4. WebView2 Runtime (normally already present on supported Windows versions)

Install Wails:

```powershell
go install github.com/wailsapp/wails/v2/cmd/wails@v2.15.0
```

Make sure Go's bin directory is on your PATH.

Check:

```powershell
go version
node --version
npm --version
wails version
```

## Build

Open PowerShell or Command Prompt in the project directory:

```powershell
wails build
```

Or:

```powershell
build-windows.bat
```

The executable will normally be:

```text
build\bin\PortMonitor.exe
```

Copy that executable to a Windows machine and launch it.

## Development

Wails generates the Go-to-JavaScript bindings during the normal Wails build/dev flow. If you want to install frontend dependencies manually:

```powershell
cd frontend
npm install
cd ..
```

Run:

```powershell
wails dev
```

## How administrator elevation works

The normal scan does not require administrator privileges.

When a process cannot be terminated because Windows denies access:

1. Port Monitor explains that administrator privileges are required.
2. The user can choose **Run as Administrator**.
3. Windows displays the standard UAC prompt.
4. A new elevated PortMonitor process starts.
5. The old non-elevated process exits.

The application does not implement a password field and does not collect or store Windows credentials.

## Windows behavior and limitations

Some system services and protected processes cannot be terminated even by a normal application. Windows may reject those operations.

The application reports the Windows error rather than silently claiming success.

A port may disappear after the initial scan because its process exited or stopped listening. The kill operation re-checks the PID and handles that situation.

IPv4 and IPv6 listeners are both parsed because `netstat -ano -p tcp` can return both forms.

A single process can listen on multiple ports, so each port/PID combination is displayed separately.

## Security notes

The application intentionally uses process termination only after an explicit user confirmation.

`taskkill` is invoked with an explicit PID rather than a process-name wildcard. This reduces the chance of accidentally terminating unrelated processes with the same executable name.

Do not add a password field or attempt to automate UAC credential entry. Windows owns the authentication prompt.

## Project structure

```text
PortMonitor/
├── app.go
├── main.go
├── go.mod
├── wails.json
├── build-windows.bat
├── README.md
└── frontend/
    ├── index.html
    ├── package.json
    └── src/
        ├── main.js
        └── style.css
```

## Important note about the generated executable

This repository is source-complete, but the final Windows `.exe` should be built on Windows because Wails uses Windows desktop/WebView components. The provided build script produces:

```text
build\bin\PortMonitor.exe
```
