package system

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

// Service represents a detected running service or group of identical processes.
type Service struct {
	Name       string `json:"name"`
	PID        int    `json:"pid"`
	Executable string `json:"executable"`
	CmdLine    string `json:"cmdline"`
	WorkingDir string `json:"working_dir"`
	Instances  int    `json:"instances"`
	Language   string `json:"language,omitempty"`
	User       string `json:"user,omitempty"`
}

// ServiceCache stores discovered services in a thread-safe map.
type ServiceCache struct {
	m sync.Map
}

func (c *ServiceCache) Store(key string, svc Service) {
	c.m.Store(key, svc)
}

func (c *ServiceCache) Load(key string) (Service, bool) {
	v, ok := c.m.Load(key)
	if !ok {
		return Service{}, false
	}
	return v.(Service), true
}

// All returns a snapshot of all cached services.
func (c *ServiceCache) All() []Service {
	var services []Service
	c.m.Range(func(_, v any) bool {
		services = append(services, v.(Service))
		return true
	})
	return services
}

// processInfo holds raw data read from /proc/[pid]/.
type processInfo struct {
	PID        int
	Executable string
	CmdLine    []string
	WorkingDir string
	Environ    map[string]string
	Name       string
	UID        string
}

// systemExcludeList contains process names that are well-known OS/shell
// utilities and not application services. Skipping these reduces noise.
var systemExcludeList = map[string]bool{
	"bash": true, "sh": true, "dash": true, "zsh": true, "fish": true,
	"kworker": true, "kthreadd": true, "ksoftirqd": true, "migration": true,
	"rcu_sched": true, "rcu_bh": true, "watchdog": true, "cpuhp": true,
	"kdevtmpfs": true, "netns": true, "kauditd": true, "khungtaskd": true,
	"oom_reaper": true, "writeback": true, "kcompactd0": true,
	"kintegrityd": true, "crypto": true, "kblockd": true, "kswapd0": true,
	"vmstat": true, "khugepaged": true, "kthrotld": true, "ipv6_addrconf": true,
	"ps": true, "grep": true, "top": true, "htop": true, "ss": true,
	"ls": true, "cat": true, "head": true, "tail": true, "awk": true, "sed": true,
	"systemd": true, "systemd-journal": true, "systemd-udevd": true,
	"systemd-resolved": true, "systemd-networkd": true, "systemd-logind": true,
	"agetty": true, "login": true, "sshd": true, "cron": true, "atd": true,
	"dbus-daemon": true, "polkitd": true, "rsyslogd": true, "acpid": true,
	"NetworkManager": true, "wpa_supplicant": true, "avahi-daemon": true,
	"bluetoothd": true, "upowerd": true, "udisksd": true, "snapd": true,
}

// DiscoverServices scans /proc and returns all detected application services
// grouped by their executable name.
func DiscoverServices() ([]Service, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, fmt.Errorf("reading /proc: %w", err)
	}

	selfPID := os.Getpid()

	var (
		mu  sync.Mutex
		wg  sync.WaitGroup
		sem = make(chan struct{}, 10)
		raw []processInfo
	)

	for _, entry := range entries {
		pid, err := strconv.Atoi(entry.Name())
		if err != nil || pid == selfPID || pid == 1 {
			continue
		}

		wg.Add(1)
		sem <- struct{}{}
		go func(pid int) {
			defer wg.Done()
			defer func() { <-sem }()

			info, err := readProcessInfo(pid)
			if err != nil || info == nil {
				return
			}

			mu.Lock()
			raw = append(raw, *info)
			mu.Unlock()
		}(pid)
	}
	wg.Wait()

	// Group by comm name (basename of exe) so worker forks share one entry.
	groups := make(map[string][]processInfo)
	for _, p := range raw {
		key := p.Name
		if key == "" {
			key = filepath.Base(p.Executable)
		}
		groups[key] = append(groups[key], p)
	}

	var services []Service
	for name, group := range groups {
		if systemExcludeList[name] {
			continue
		}

		// Use the first (lowest-PID) process as the representative.
		rep := group[0]
		lang := DetectLanguage(rep)

		services = append(services, Service{
			Name:       name,
			PID:        rep.PID,
			Executable: rep.Executable,
			CmdLine:    strings.Join(rep.CmdLine, " "),
			WorkingDir: rep.WorkingDir,
			Instances:  len(group),
			Language:   lang,
			User:       rep.UID,
		})
	}

	log.Printf("discovered %d services from %d processes", len(services), len(raw))
	return services, nil
}

// readProcessInfo reads all process metadata from /proc/[pid]/.
// Returns nil without error for kernel threads (no exe link).
func readProcessInfo(pid int) (*processInfo, error) {
	base := fmt.Sprintf("/proc/%d", pid)

	exePath, err := os.Readlink(filepath.Join(base, "exe"))
	if err != nil {
		// Kernel threads have no exe symlink – silently skip.
		return nil, nil //nolint:nilerr
	}

	name := readComm(base)
	if name == "" {
		name = filepath.Base(exePath)
	}

	return &processInfo{
		PID:        pid,
		Executable: exePath,
		CmdLine:    readCmdLine(base),
		WorkingDir: readCwd(base),
		Environ:    readEnviron(base),
		Name:       name,
		UID:        readUID(base),
	}, nil
}

func readComm(base string) string {
	data, err := os.ReadFile(filepath.Join(base, "comm"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func readCmdLine(base string) []string {
	data, err := os.ReadFile(filepath.Join(base, "cmdline"))
	if err != nil || len(data) == 0 {
		return nil
	}
	// /proc/[pid]/cmdline uses NUL bytes as argument separators.
	var parts []string
	for _, p := range strings.Split(string(data), "\x00") {
		if p = strings.TrimSpace(p); p != "" {
			parts = append(parts, p)
		}
	}
	return parts
}

func readCwd(base string) string {
	cwd, _ := os.Readlink(filepath.Join(base, "cwd"))
	return cwd
}

func readEnviron(base string) map[string]string {
	data, err := os.ReadFile(filepath.Join(base, "environ"))
	if err != nil || len(data) == 0 {
		return nil
	}
	env := make(map[string]string)
	// /proc/[pid]/environ uses NUL bytes as variable separators.
	for _, pair := range strings.Split(string(data), "\x00") {
		if idx := strings.IndexByte(pair, '='); idx > 0 {
			env[pair[:idx]] = pair[idx+1:]
		}
	}
	return env
}

func readUID(base string) string {
	f, err := os.Open(filepath.Join(base, "status"))
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "Uid:") {
			if fields := strings.Fields(line); len(fields) >= 2 {
				return fields[1] // real UID
			}
		}
	}
	return ""
}
