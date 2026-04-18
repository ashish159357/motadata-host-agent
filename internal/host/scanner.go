package host

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

func Scan(ctx context.Context, procRoot string, skipPID int) (ScanResult, error) {
	entries, err := os.ReadDir(procRoot)
	if err != nil {
		return ScanResult{}, fmt.Errorf("read proc root %q: %w", procRoot, err)
	}

	hostname, _ := os.Hostname()
	services := make([]ServiceRecord, 0, len(entries))

	for _, entry := range entries {
		select {
		case <-ctx.Done():
			return ScanResult{}, ctx.Err()
		default:
		}

		if !entry.IsDir() {
			continue
		}

		pid, err := strconv.Atoi(entry.Name())
		if err != nil || pid <= 0 || pid == skipPID {
			continue
		}

		info, err := readProcessInfo(procRoot, pid)
		if err != nil {
			continue
		}

		detection := DetectLanguage(info)
		services = append(services, buildServiceRecord(info, detection))
	}

	sort.Slice(services, func(i, j int) bool {
		if services[i].PID == services[j].PID {
			return services[i].Name < services[j].Name
		}
		return services[i].PID < services[j].PID
	})

	detected := 0
	for _, service := range services {
		if service.Detection.Language != "" && service.Detection.Language != "unknown" {
			detected++
		}
	}

	return ScanResult{
		Host: HostInfo{
			Hostname:  hostname,
			ProcRoot:  procRoot,
			ScannedAt: time.Now().UTC(),
		},
		Summary: Summary{
			TotalProcesses:    len(services),
			DetectedProcesses: detected,
			UnknownProcesses:  len(services) - detected,
		},
		Services: services,
	}, nil
}

func buildServiceRecord(info ProcessInfo, detection Detection) ServiceRecord {
	command := strings.Join(info.Cmdline, " ")
	name := info.Name
	if name == "" {
		name = deriveProcessName(info)
	}

	return ServiceRecord{
		PID:        info.PID,
		PPID:       info.PPID,
		UID:        info.UID,
		Name:       name,
		Command:    command,
		Executable: info.Executable,
		CWD:        info.CWD,
		Detection:  detection,
	}
}

func readProcessInfo(procRoot string, pid int) (ProcessInfo, error) {
	info := ProcessInfo{PID: pid}
	base := filepath.Join(procRoot, strconv.Itoa(pid))

	if status, err := os.ReadFile(filepath.Join(base, "status")); err == nil {
		parseStatus(status, &info)
	}

	if cmdline, err := os.ReadFile(filepath.Join(base, "cmdline")); err == nil {
		info.Cmdline = parseCmdline(cmdline)
	}

	if comm, err := os.ReadFile(filepath.Join(base, "comm")); err == nil {
		info.Comm = strings.TrimSpace(string(comm))
	}

	if exe, err := os.Readlink(filepath.Join(base, "exe")); err == nil {
		info.Executable = exe
	}

	if cwd, err := os.Readlink(filepath.Join(base, "cwd")); err == nil {
		info.CWD = cwd
	}

	if environ, err := os.ReadFile(filepath.Join(base, "environ")); err == nil {
		info.Env = parseEnviron(environ)
	}

	if info.Name == "" {
		info.Name = deriveProcessName(info)
	}

	if len(info.Cmdline) == 0 && info.Executable == "" && info.CWD == "" {
		return ProcessInfo{}, fmt.Errorf("insufficient process metadata")
	}

	if info.Name == "" {
		return ProcessInfo{}, fmt.Errorf("insufficient process metadata")
	}

	if info.Comm == "" {
		info.Comm = info.Name
	}

	return info, nil
}

func parseStatus(data []byte, info *ProcessInfo) {
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, "Name:"):
			info.Name = strings.TrimSpace(strings.TrimPrefix(line, "Name:"))
		case strings.HasPrefix(line, "PPid:"):
			if v, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, "PPid:"))); err == nil {
				info.PPID = v
			}
		case strings.HasPrefix(line, "Uid:"):
			fields := strings.Fields(strings.TrimSpace(strings.TrimPrefix(line, "Uid:")))
			if len(fields) > 0 {
				if v, err := strconv.Atoi(fields[0]); err == nil {
					info.UID = v
				}
			}
		}
	}
}

func parseCmdline(data []byte) []string {
	raw := strings.Split(string(data), "\x00")
	args := make([]string, 0, len(raw))
	for _, arg := range raw {
		if arg = strings.TrimSpace(arg); arg != "" {
			args = append(args, arg)
		}
	}
	return args
}

func parseEnviron(data []byte) map[string]string {
	env := make(map[string]string)
	for _, item := range strings.Split(string(data), "\x00") {
		if item == "" {
			continue
		}
		key, value, found := strings.Cut(item, "=")
		if !found {
			continue
		}
		env[key] = value
	}
	return env
}

func deriveProcessName(info ProcessInfo) string {
	if len(info.Cmdline) > 0 {
		base := filepath.Base(info.Cmdline[0])
		if base != "." && base != string(filepath.Separator) {
			return base
		}
	}

	if info.Executable != "" {
		return filepath.Base(info.Executable)
	}

	if info.Comm != "" {
		return info.Comm
	}

	return ""
}
