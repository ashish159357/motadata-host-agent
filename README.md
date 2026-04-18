# Motadata Host Agent

Bare-metal host agent that scans Linux processes under `/proc`, identifies the most likely programming language for each process, and emits a structured JSON report.

## Project Layout

- `cmd/motadata-host-agent`: executable entrypoint
- `internal/agent`: orchestration and JSON output
- `internal/config`: environment-based configuration loading
- `internal/host`: process scanning and language detection

## What It Detects

The scanner inspects host processes and uses a layered set of heuristics:

- command-line interpreter names and script extensions
- environment variable hints
- working directory and executable-path file markers
- executable basename fallbacks

Detected languages currently include:

- `python`
- `nodejs`
- `dotnet`
- `go`
- `ruby`
- `php`
- `perl`
- `rust`
- `unknown`

## Configuration

| Variable | Description | Default |
| --- | --- | --- |
| `PROC_ROOT` | Path to the proc filesystem root | `/proc` |
| `OUTPUT_FILE` | Optional file to write the JSON report to | empty |
| `PRETTY_JSON` | Pretty-print the JSON output | `true` |

## Run

```bash
go run ./cmd/motadata-host-agent
```

Write the JSON output to a file:

```bash
OUTPUT_FILE=/tmp/motadata-host-scan.json go run ./cmd/motadata-host-agent
```

Use a different proc root for offline testing:

```bash
PROC_ROOT=/path/to/fake/proc go run ./cmd/motadata-host-agent
```

## Test

```bash
go test ./...
go build ./...
```

## JSON Output Shape

```json
{
  "host": {
    "hostname": "server-01",
    "proc_root": "/proc",
    "scanned_at": "2026-04-17T10:00:00Z"
  },
  "summary": {
    "total_processes": 12,
    "detected_processes": 8,
    "unknown_processes": 4
  },
  "services": [
    {
      "pid": 1234,
      "ppid": 1,
      "uid": 1000,
      "name": "python3",
      "command": "python3 app.py",
      "executable": "/usr/bin/python3.11",
      "cwd": "/srv/app",
      "detection": {
        "language": "python",
        "method": "cmdline",
        "confidence": "high",
        "evidence": ["python3", "app.py"]
      }
    }
  ]
}
```
