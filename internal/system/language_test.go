package system

import (
	"testing"
)

func TestDetectFromEnviron(t *testing.T) {
	cases := []struct {
		name     string
		env      map[string]string
		expected string
	}{
		{
			name:     "dotnet ASPNETCORE",
			env:      map[string]string{"ASPNETCORE_ENVIRONMENT": "Production"},
			expected: "dotnet",
		},
		{
			name:     "dotnet DOTNET prefix",
			env:      map[string]string{"DOTNET_RUNNING_IN_CONTAINER": "true"},
			expected: "dotnet",
		},
		{
			name:     "java JAVA_TOOL_OPTIONS",
			env:      map[string]string{"JAVA_TOOL_OPTIONS": "-Xmx512m"},
			expected: "java",
		},
		{
			name:     "java CATALINA",
			env:      map[string]string{"CATALINA_HOME": "/opt/tomcat"},
			expected: "java",
		},
		{
			name:     "java JAVA_VERSION env var",
			env:      map[string]string{"JAVA_VERSION": "11.0.21"},
			expected: "java",
		},
		{
			name:     "java CLASSPATH env var",
			env:      map[string]string{"CLASSPATH": "/opt/app/lib/*"},
			expected: "java",
		},
		{
			name:     "java JDK_JAVA_OPTIONS env var",
			env:      map[string]string{"JDK_JAVA_OPTIONS": "--add-opens java.base/java.lang=ALL-UNNAMED"},
			expected: "java",
		},
		{
			name:     "go GOMEMLIMIT",
			env:      map[string]string{"GOMEMLIMIT": "512MiB"},
			expected: "go",
		},
		{
			name:     "go GOPATH env var",
			env:      map[string]string{"GOPATH": "/home/user/go"},
			expected: "go",
		},
		{
			name:     "go GOFLAGS env var",
			env:      map[string]string{"GOFLAGS": "-mod=vendor"},
			expected: "go",
		},
		{
			name:     "go GONOSUMDB env var",
			env:      map[string]string{"GONOSUMDB": "*.internal.example.com"},
			expected: "go",
		},
		{
			name:     "python PYTHONPATH",
			env:      map[string]string{"PYTHONPATH": "/usr/lib/python3"},
			expected: "python",
		},
		{
			name:     "python VIRTUAL_ENV",
			env:      map[string]string{"VIRTUAL_ENV": "/home/user/venv"},
			expected: "python",
		},
		{
			name:     "nodejs NODE_ENV",
			env:      map[string]string{"NODE_ENV": "production"},
			expected: "nodejs",
		},
		{
			name:     "ruby RAILS_ENV",
			env:      map[string]string{"RAILS_ENV": "production"},
			expected: "ruby",
		},
		{
			name:     "php PHPRC",
			env:      map[string]string{"PHPRC": "/etc/php/8.1"},
			expected: "php",
		},
		{
			name:     "rust CARGO_HOME",
			env:      map[string]string{"CARGO_HOME": "/home/user/.cargo"},
			expected: "rust",
		},
		{
			name:     "unknown env",
			env:      map[string]string{"PATH": "/usr/bin", "HOME": "/root"},
			expected: "",
		},
		{
			name:     "nil env",
			env:      nil,
			expected: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := detectFromEnviron(tc.env)
			if got != tc.expected {
				t.Errorf("detectFromEnviron(%v) = %q, want %q", tc.env, got, tc.expected)
			}
		})
	}
}

func TestDetectFromCmdLine(t *testing.T) {
	cases := []struct {
		name     string
		cmdline  []string
		exePath  string
		expected string
	}{
		{
			name:     "java interpreter",
			cmdline:  []string{"java", "-jar", "app.jar"},
			exePath:  "/usr/bin/java",
			expected: "java",
		},
		{
			name:     "jar flag",
			cmdline:  []string{"/opt/myapp/bin/myapp", "-jar", "app.jar"},
			exePath:  "/opt/myapp/bin/myapp",
			expected: "java",
		},
		{
			name:     "node interpreter",
			cmdline:  []string{"node", "server.js"},
			exePath:  "/usr/bin/node",
			expected: "nodejs",
		},
		{
			name:     "js file argument",
			cmdline:  []string{"/usr/bin/node", "index.js"},
			exePath:  "/usr/bin/node",
			expected: "nodejs",
		},
		{
			name:     "python3 interpreter",
			cmdline:  []string{"python3", "app.py"},
			exePath:  "/usr/bin/python3",
			expected: "python",
		},
		{
			name:     "python versioned",
			cmdline:  []string{"python3.11", "manage.py", "runserver"},
			exePath:  "/usr/bin/python3.11",
			expected: "python",
		},
		{
			name:     "py file argument",
			cmdline:  []string{"/usr/bin/python3", "worker.py"},
			exePath:  "/usr/bin/python3",
			expected: "python",
		},
		{
			name:     "dotnet",
			cmdline:  []string{"dotnet", "myapp.dll"},
			exePath:  "/usr/bin/dotnet",
			expected: "dotnet",
		},
		{
			name:     "ruby interpreter",
			cmdline:  []string{"ruby", "app.rb"},
			exePath:  "/usr/bin/ruby",
			expected: "ruby",
		},
		{
			name:     "php interpreter",
			cmdline:  []string{"php", "-f", "index.php"},
			exePath:  "/usr/bin/php",
			expected: "php",
		},
		{
			name:     "php-fpm",
			cmdline:  []string{"php-fpm8.1"},
			exePath:  "/usr/sbin/php-fpm8.1",
			expected: "php",
		},
		{
			name:     "java .jar file argument",
			cmdline:  []string{"/usr/bin/java", "myapp.jar"},
			exePath:  "/usr/bin/java",
			expected: "java",
		},
		{
			name:     "go .go file argument",
			cmdline:  []string{"/usr/local/go/bin/go", "run", "main.go"},
			exePath:  "/usr/local/go/bin/go",
			expected: "go",
		},
		{
			name:     "unknown binary",
			cmdline:  []string{"/usr/sbin/nginx", "-g", "daemon off;"},
			exePath:  "/usr/sbin/nginx",
			expected: "",
		},
		{
			name:     "empty",
			cmdline:  nil,
			exePath:  "",
			expected: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := detectFromCmdLine(tc.cmdline, tc.exePath)
			if got != tc.expected {
				t.Errorf("detectFromCmdLine(%v, %q) = %q, want %q", tc.cmdline, tc.exePath, got, tc.expected)
			}
		})
	}
}

func TestClassifyByFiles(t *testing.T) {
	cases := []struct {
		name     string
		files    []string
		expected string
	}{
		{
			name:     "dotnet appsettings",
			files:    []string{"appsettings.json", "myapp.dll", "myapp.pdb"},
			expected: "dotnet",
		},
		{
			name:     "dotnet deps.json",
			files:    []string{"myapp.deps.json", "myapp.runtimeconfig.json"},
			expected: "dotnet",
		},
		{
			name:     "nodejs package.json",
			files:    []string{"package.json", "index.js", "node_modules"},
			expected: "nodejs",
		},
		{
			name:     "java pom.xml",
			files:    []string{"pom.xml", "src", "target"},
			expected: "java",
		},
		{
			name:     "java settings.gradle",
			files:    []string{"settings.gradle", "build.gradle", "gradlew"},
			expected: "java",
		},
		{
			name:     "java gradlew wrapper only",
			files:    []string{"gradlew", "gradle.properties"},
			expected: "java",
		},
		{
			name:     "java jar",
			files:    []string{"app.jar", "config.yml"},
			expected: "java",
		},
		{
			name:     "python requirements",
			files:    []string{"requirements.txt", "main.py", "app.py"},
			expected: "python",
		},
		{
			name:     "python pyproject",
			files:    []string{"pyproject.toml", "src"},
			expected: "python",
		},
		{
			name:     "go mod",
			files:    []string{"go.mod", "go.sum", "main.go"},
			expected: "go",
		},
		{
			name:     "go go.work workspace",
			files:    []string{"go.work", "go.work.sum"},
			expected: "go",
		},
		{
			name:     "go .go source file",
			files:    []string{"main.go", "README.md"},
			expected: "go",
		},
		{
			name:     "ruby gemfile",
			files:    []string{"gemfile", "gemfile.lock", "config.ru"},
			expected: "ruby",
		},
		{
			name:     "php composer",
			files:    []string{"composer.json", "composer.lock", "index.php"},
			expected: "php",
		},
		{
			name:     "rust cargo",
			files:    []string{"cargo.toml", "cargo.lock", "src"},
			expected: "rust",
		},
		{
			name:     "cpp cmake with sources",
			files:    []string{"cmakelists.txt", "main.cpp", "include"},
			expected: "cpp",
		},
		{
			name:     "cmake without sources",
			files:    []string{"cmakelists.txt", "README.md"},
			expected: "",
		},
		{
			name:     "empty directory",
			files:    []string{},
			expected: "",
		},
		{
			name:     "unknown files",
			files:    []string{"README.md", "LICENSE", "Makefile"},
			expected: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := classifyByFiles(tc.files)
			if got != tc.expected {
				t.Errorf("classifyByFiles(%v) = %q, want %q", tc.files, got, tc.expected)
			}
		})
	}
}

func TestDetectFromExecutablePath(t *testing.T) {
	cases := []struct {
		name     string
		exePath  string
		expected string
	}{
		{
			name:     "openjdk path",
			exePath:  "/usr/lib/jvm/java-11-openjdk-amd64/bin/java",
			expected: "java",
		},
		{
			name:     "java java- prefix path",
			exePath:  "/usr/lib/jvm/java-17/bin/java",
			expected: "java",
		},
		{
			name:     "python path",
			exePath:  "/usr/bin/python3.11",
			expected: "python",
		},
		{
			name:     "pyenv path",
			exePath:  "/home/user/.pyenv/versions/3.11.0/bin/python",
			expected: "python",
		},
		{
			name:     "node path",
			exePath:  "/usr/local/node/bin/node",
			expected: "nodejs",
		},
		{
			name:     "dotnet path",
			exePath:  "/usr/share/dotnet/dotnet",
			expected: "dotnet",
		},
		{
			name:     "go bin",
			exePath:  "/home/user/go/bin/myapp",
			expected: "go",
		},
		{
			name:     "go usr/local/go path",
			exePath:  "/usr/local/go/bin/go",
			expected: "go",
		},
		{
			name:     "rbenv path",
			exePath:  "/home/user/.rbenv/versions/3.2.0/bin/ruby",
			expected: "ruby",
		},
		{
			name:     "cargo bin",
			exePath:  "/home/user/.cargo/bin/myapp",
			expected: "rust",
		},
		{
			name:     "nginx - unknown",
			exePath:  "/usr/sbin/nginx",
			expected: "",
		},
		{
			name:     "empty path",
			exePath:  "",
			expected: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := detectFromExecutablePath(tc.exePath)
			if got != tc.expected {
				t.Errorf("detectFromExecutablePath(%q) = %q, want %q", tc.exePath, got, tc.expected)
			}
		})
	}
}
