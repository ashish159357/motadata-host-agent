package system

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// DetectLanguage runs a multi-stage pipeline to identify the programming
// language of a process. Stages are ordered from most to least reliable.
func DetectLanguage(proc processInfo) string {
	if lang := detectFromEnviron(proc.Environ); lang != "" {
		log.Printf("pid %d (%s): language=%s (environ)", proc.PID, proc.Name, lang)
		return lang
	}

	if lang := detectFromCmdLine(proc.CmdLine, proc.Executable); lang != "" {
		log.Printf("pid %d (%s): language=%s (cmdline)", proc.PID, proc.Name, lang)
		return lang
	}

	if lang := detectFromFiles(proc.WorkingDir); lang != "" {
		log.Printf("pid %d (%s): language=%s (files in %s)", proc.PID, proc.Name, lang, proc.WorkingDir)
		return lang
	}

	if lang := detectFromMaps(proc.PID); lang != "" {
		log.Printf("pid %d (%s): language=%s (shared libs)", proc.PID, proc.Name, lang)
		return lang
	}

	if lang := detectFromExecutablePath(proc.Executable); lang != "" {
		log.Printf("pid %d (%s): language=%s (exe path)", proc.PID, proc.Name, lang)
		return lang
	}

	return ""
}

// detectFromEnviron maps well-known environment variables to their languages.
// Environment variables are the most reliable indicator because runtimes set
// them deliberately (e.g. JVM sets JAVA_TOOL_OPTIONS, .NET sets ASPNETCORE_*).
func detectFromEnviron(env map[string]string) string {
	if len(env) == 0 {
		return ""
	}

	for key := range env {
		switch {
		case strings.HasPrefix(key, "ASPNETCORE_") || strings.HasPrefix(key, "DOTNET_"):
			return "dotnet"
		case key == "JAVA_TOOL_OPTIONS" || key == "JAVA_OPTS" ||
			strings.HasPrefix(key, "CATALINA_") || key == "JVM_OPTS" ||
			key == "JAVA_HOME" || key == "JAVA_VERSION" ||
			key == "CLASSPATH" || key == "JDK_JAVA_OPTIONS":
			return "java"
		case key == "GOMEMLIMIT" || key == "GOGC" || key == "GOMAXPROCS" ||
			key == "GOPATH" || key == "GOFLAGS" || key == "GONOSUMDB" || key == "GONOSUMCHECK":
			return "go"
		case key == "PYTHONPATH" || key == "PYTHONHOME" ||
			key == "VIRTUAL_ENV" || strings.HasPrefix(key, "CONDA_"):
			return "python"
		case key == "NODE_ENV" || key == "NODE_OPTIONS" ||
			key == "NODE_PATH" || key == "NPM_CONFIG_PREFIX":
			return "nodejs"
		case key == "RAILS_ENV" || key == "RACK_ENV" || key == "BUNDLE_PATH":
			return "ruby"
		case key == "PHP_INI_SCAN_DIR" || key == "PHPRC":
			return "php"
		case key == "CARGO_HOME" || key == "RUSTUP_HOME":
			return "rust"
		}
	}
	return ""
}

// detectFromCmdLine checks the process binary name and arguments.
// Interpreter names (java, node, python…) are a strong indicator because
// scripts are invoked as `python app.py` or `java -jar app.jar`.
func detectFromCmdLine(cmdline []string, exePath string) string {
	candidates := append([]string{exePath}, cmdline...)

	for _, arg := range candidates {
		base := strings.ToLower(filepath.Base(arg))

		switch base {
		case "java", "javaw":
			return "java"
		case "node", "nodejs", "npm", "npx", "yarn", "pnpm":
			return "nodejs"
		case "python", "python2", "python3":
			return "python"
		case "dotnet":
			return "dotnet"
		case "ruby", "bundle":
			return "ruby"
		case "php", "php-fpm":
			return "php"
		case "perl", "perl5":
			return "perl"
		case "go":
			return "go"
		}

		if strings.HasPrefix(base, "python") {
			return "python"
		}
		if strings.HasPrefix(base, "php") && !strings.Contains(base, "phpstorm") {
			return "php"
		}
		if strings.HasPrefix(base, "ruby") {
			return "ruby"
		}
	}

	// Check argument extensions / flags.
	for _, arg := range cmdline {
		lower := strings.ToLower(arg)
		switch {
		case strings.HasSuffix(lower, ".py"):
			return "python"
		case strings.HasSuffix(lower, ".js") || strings.HasSuffix(lower, ".mjs"):
			return "nodejs"
		case strings.HasSuffix(lower, ".rb"):
			return "ruby"
		case strings.HasSuffix(lower, ".php"):
			return "php"
		case lower == "-jar" || strings.HasSuffix(lower, ".jar"):
			return "java"
		case strings.HasSuffix(lower, ".go"):
			return "go"
		}
	}

	return ""
}

// detectFromFiles inspects the working directory for language-specific marker
// files (package.json, pom.xml, requirements.txt, …).  This stage catches
// compiled binaries that have no interpreter in their command line.
func detectFromFiles(dir string) string {
	if dir == "" || dir == "/" {
		return ""
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}

	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, strings.ToLower(e.Name()))
	}

	return classifyByFiles(names)
}

// classifyByFiles maps a set of filenames to a language.
// Rules are ordered from most specific (unique marker) to least specific.
func classifyByFiles(files []string) string {
	set := make(map[string]bool, len(files))
	for _, f := range files {
		set[f] = true
	}

	// dotnet
	if set["appsettings.json"] || set["web.config"] {
		return "dotnet"
	}
	for _, f := range files {
		if strings.HasSuffix(f, ".pdb") || strings.HasSuffix(f, ".deps.json") {
			return "dotnet"
		}
	}

	// nodejs
	if set["package.json"] || set["node_modules"] {
		return "nodejs"
	}

	// java
	if set["pom.xml"] || set["build.gradle"] || set["build.gradle.kts"] ||
		set["settings.gradle"] || set["gradlew"] {
		return "java"
	}
	for _, f := range files {
		if strings.HasSuffix(f, ".jar") || strings.HasSuffix(f, ".war") || strings.HasSuffix(f, ".ear") {
			return "java"
		}
	}

	// python
	if set["requirements.txt"] || set["setup.py"] || set["pyproject.toml"] ||
		set["setup.cfg"] || set["pipfile"] || set["poetry.lock"] {
		return "python"
	}
	for _, f := range files {
		if strings.HasSuffix(f, ".py") {
			return "python"
		}
	}

	// go
	if set["go.mod"] || set["go.sum"] || set["go.work"] || set["go.work.sum"] {
		return "go"
	}
	for _, f := range files {
		if strings.HasSuffix(f, ".go") {
			return "go"
		}
	}

	// ruby
	if set["gemfile"] || set["gemfile.lock"] || set[".ruby-version"] {
		return "ruby"
	}
	for _, f := range files {
		if strings.HasSuffix(f, ".rb") || strings.HasSuffix(f, ".gemspec") {
			return "ruby"
		}
	}

	// php
	if set["composer.json"] || set["composer.lock"] {
		return "php"
	}
	for _, f := range files {
		if strings.HasSuffix(f, ".php") {
			return "php"
		}
	}

	// rust
	if set["cargo.toml"] || set["cargo.lock"] {
		return "rust"
	}

	// cpp (only when a build system is present alongside source files)
	if set["cmakelists.txt"] || set["meson.build"] || set["makefile"] {
		for _, f := range files {
			if strings.HasSuffix(f, ".cpp") || strings.HasSuffix(f, ".cc") ||
				strings.HasSuffix(f, ".cxx") || strings.HasSuffix(f, ".hpp") {
				return "cpp"
			}
		}
	}

	return ""
}

// detectFromMaps reads /proc/[pid]/maps and looks for known runtime shared
// libraries.  This is useful for compiled languages (Go, C++) and embedded
// runtimes (JVM embedded in a native launcher, Python C extensions, etc.).
func detectFromMaps(pid int) string {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/maps", pid))
	if err != nil {
		return ""
	}

	content := strings.ToLower(string(data))

	switch {
	case strings.Contains(content, "libjvm.so") || strings.Contains(content, "libjava.so"):
		return "java"
	case strings.Contains(content, "libpython"):
		return "python"
	case strings.Contains(content, "libnode.so"):
		return "nodejs"
	case strings.Contains(content, "libmono") || strings.Contains(content, "libcoreclr"):
		return "dotnet"
	case strings.Contains(content, "libruby"):
		return "ruby"
	case strings.Contains(content, "libphp"):
		return "php"
	case strings.Contains(content, "libstdc++.so") || strings.Contains(content, "libgcc_s.so"):
		// libstdc++ is used by many languages; only flag as cpp as a last resort.
		return "cpp"
	}

	return ""
}

// detectFromExecutablePath infers language from well-known installation paths
// (e.g. /usr/lib/jvm/…, /usr/bin/python3, ~/.pyenv/…).
func detectFromExecutablePath(exePath string) string {
	if exePath == "" {
		return ""
	}

	lower := strings.ToLower(exePath)

	switch {
	case strings.Contains(lower, "jdk") || strings.Contains(lower, "jre") ||
		strings.Contains(lower, "openjdk") || strings.Contains(lower, "java-"):
		return "java"
	case strings.Contains(lower, "python") || strings.Contains(lower, "pyenv"):
		return "python"
	case strings.Contains(lower, "nodejs") || strings.Contains(lower, "/node/bin"):
		return "nodejs"
	case strings.Contains(lower, "dotnet"):
		return "dotnet"
	case strings.Contains(lower, "/go/bin/") || strings.Contains(lower, "golang") ||
		strings.Contains(lower, "/usr/local/go/"):
		return "go"
	case strings.Contains(lower, "ruby") || strings.Contains(lower, "rbenv") ||
		strings.Contains(lower, "/rvm/"):
		return "ruby"
	case strings.Contains(lower, "/php") || strings.Contains(lower, "phpenv"):
		return "php"
	case strings.Contains(lower, "rustup") || strings.Contains(lower, "/.cargo/"):
		return "rust"
	}

	return ""
}
