package host

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestScanDetectsMultipleProcesses(t *testing.T) {
	procRoot := t.TempDir()

	pythonApp := t.TempDir()
	writeFile(t, filepath.Join(pythonApp, "requirements.txt"), "flask\n")

	nodeApp := t.TempDir()
	writeFile(t, filepath.Join(nodeApp, "package.json"), "{}\n")

	javaApp := t.TempDir()

	createProcess(t, procRoot, 1001, processFixture{
		ppid:       1,
		uid:        1000,
		name:       "python",
		cmdline:    []string{"python3", "app.py"},
		executable: "/usr/bin/python3.11",
		cwd:        pythonApp,
	})

	createProcess(t, procRoot, 1002, processFixture{
		ppid:       1,
		uid:        1001,
		name:       "node",
		cmdline:    []string{"node", "server.js"},
		executable: "/usr/bin/node",
		cwd:        nodeApp,
	})

	createProcess(t, procRoot, 1003, processFixture{
		ppid:       1,
		uid:        1002,
		name:       "java",
		cmdline:    []string{"java", "-jar", "service.jar"},
		executable: "/usr/bin/java",
		cwd:        javaApp,
	})

	result, err := Scan(context.Background(), procRoot, -1)
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	if got, want := result.Summary.TotalProcesses, 3; got != want {
		t.Fatalf("total processes = %d, want %d", got, want)
	}

	if got, want := result.Summary.DetectedProcesses, 2; got != want {
		t.Fatalf("detected processes = %d, want %d", got, want)
	}

	if len(result.Services) != 3 {
		t.Fatalf("services length = %d, want 3", len(result.Services))
	}

	if result.Services[0].Detection.Language != "python" {
		t.Fatalf("service[0] language = %q, want python", result.Services[0].Detection.Language)
	}
	if result.Services[1].Detection.Language != "nodejs" {
		t.Fatalf("service[1] language = %q, want nodejs", result.Services[1].Detection.Language)
	}
	if result.Services[2].Detection.Language != "unknown" {
		t.Fatalf("service[2] language = %q, want unknown", result.Services[2].Detection.Language)
	}
}

type processFixture struct {
	ppid       int
	uid        int
	name       string
	cmdline    []string
	executable string
	cwd        string
	env        map[string]string
}

func createProcess(t *testing.T, procRoot string, pid int, fixture processFixture) {
	t.Helper()

	pidDir := filepath.Join(procRoot, strconv.Itoa(pid))
	if err := os.MkdirAll(pidDir, 0o755); err != nil {
		t.Fatalf("mkdir pid dir: %v", err)
	}

	status := "Name:\t" + fixture.name + "\n"
	status += "State:\tS (sleeping)\n"
	status += "PPid:\t" + strconv.Itoa(fixture.ppid) + "\n"
	status += "Uid:\t" + strconv.Itoa(fixture.uid) + "\t" + strconv.Itoa(fixture.uid) + "\t" + strconv.Itoa(fixture.uid) + "\t" + strconv.Itoa(fixture.uid) + "\n"
	writeFile(t, filepath.Join(pidDir, "status"), status)
	writeFile(t, filepath.Join(pidDir, "comm"), fixture.name+"\n")

	cmdline := ""
	for _, arg := range fixture.cmdline {
		cmdline += arg + "\x00"
	}
	writeFile(t, filepath.Join(pidDir, "cmdline"), cmdline)

	if fixture.executable != "" {
		if err := os.Symlink(fixture.executable, filepath.Join(pidDir, "exe")); err != nil {
			t.Fatalf("symlink exe: %v", err)
		}
	}

	if fixture.cwd != "" {
		if err := os.Symlink(fixture.cwd, filepath.Join(pidDir, "cwd")); err != nil {
			t.Fatalf("symlink cwd: %v", err)
		}
	}

	if len(fixture.env) > 0 {
		env := ""
		for k, v := range fixture.env {
			env += k + "=" + v + "\x00"
		}
		writeFile(t, filepath.Join(pidDir, "environ"), env)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file %s: %v", path, err)
	}
}
