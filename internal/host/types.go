package host

import "time"

type ProcessInfo struct {
	PID        int
	PPID       int
	UID        int
	Name       string
	Comm       string
	Cmdline    []string
	Executable string
	CWD        string
	Env        map[string]string
}

type Detection struct {
	Language   string   `json:"language"`
	Method     string   `json:"method"`
	Confidence string   `json:"confidence"`
	Evidence   []string `json:"evidence,omitempty"`
}

type ServiceRecord struct {
	PID        int       `json:"pid"`
	PPID       int       `json:"ppid"`
	UID        int       `json:"uid"`
	Name       string    `json:"name"`
	Command    string    `json:"command,omitempty"`
	Executable string    `json:"executable,omitempty"`
	CWD        string    `json:"cwd,omitempty"`
	Detection  Detection `json:"detection"`
}

type HostInfo struct {
	Hostname  string    `json:"hostname"`
	ProcRoot  string    `json:"proc_root"`
	ScannedAt time.Time `json:"scanned_at"`
}

type Summary struct {
	TotalProcesses    int `json:"total_processes"`
	DetectedProcesses int `json:"detected_processes"`
	UnknownProcesses  int `json:"unknown_processes"`
}

type ScanResult struct {
	Host     HostInfo        `json:"host"`
	Summary  Summary         `json:"summary"`
	Services []ServiceRecord `json:"services"`
}
