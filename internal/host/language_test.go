package host

import (
	"path/filepath"
	"testing"
)

func TestDetectLanguageFromCmdline(t *testing.T) {
	tests := []struct {
		name     string
		info     ProcessInfo
		wantLang string
	}{
		{
			name: "python command",
			info: ProcessInfo{
				Cmdline: []string{"python3", "app.py"},
			},
			wantLang: "python",
		},
		{
			name: "node command",
			info: ProcessInfo{
				Cmdline: []string{"node", "server.js"},
			},
			wantLang: "nodejs",
		},
		{
			name: "dotnet dll",
			info: ProcessInfo{
				Cmdline: []string{"dotnet", "app.dll"},
			},
			wantLang: "dotnet",
		},
		{
			name: "unknown command",
			info: ProcessInfo{
				Cmdline: []string{"/usr/local/bin/my-service"},
			},
			wantLang: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectLanguage(tt.info)
			if got.Language != tt.wantLang {
				t.Fatalf("DetectLanguage().Language = %q, want %q", got.Language, tt.wantLang)
			}
		})
	}
}

func TestDetectLanguageFromEnv(t *testing.T) {
	got := DetectLanguage(ProcessInfo{
		Env: map[string]string{
			"ASPNETCORE_URLS": "http://+:8080",
		},
	})
	if got.Language != "dotnet" {
		t.Fatalf("DetectLanguage().Language = %q, want dotnet", got.Language)
	}
}

func TestDetectLanguageFromPaths(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module example.com/app\n")

	got := DetectLanguage(ProcessInfo{
		CWD: dir,
	})
	if got.Language != "go" {
		t.Fatalf("DetectLanguage().Language = %q, want go", got.Language)
	}
}

func TestDetectLanguageDoesNotIdentifyJava(t *testing.T) {
	got := DetectLanguage(ProcessInfo{
		Cmdline:    []string{"java", "-jar", "service.jar"},
		Executable: "/usr/bin/java",
		CWD:        t.TempDir(),
		Env: map[string]string{
			"JAVA_TOOL_OPTIONS": "-Xmx512m",
		},
	})

	if got.Language != "unknown" {
		t.Fatalf("DetectLanguage().Language = %q, want unknown", got.Language)
	}
}
