package host

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func DetectLanguage(info ProcessInfo) Detection {
	if match := detectFromCmdline(info.Cmdline); match.Language != "unknown" {
		return match
	}

	if match := detectFromEnv(info.Env); match.Language != "unknown" {
		return match
	}

	if match := detectFromPaths(info.CWD, info.Executable); match.Language != "unknown" {
		return match
	}

	if match := detectFromExecutable(info.Executable, info.Comm, info.Cmdline); match.Language != "unknown" {
		return match
	}

	return Detection{
		Language:   "unknown",
		Method:     "heuristic",
		Confidence: "low",
		Evidence:   []string{"no reliable runtime or filesystem signal"},
	}
}

func detectFromCmdline(args []string) Detection {
	if len(args) == 0 {
		return unknownDetection()
	}

	first := strings.ToLower(filepath.Base(args[0]))

	switch {
	case isAny(first, "python", "python2", "python3") || hasSuffixAny(args, ".py"):
		return match("python", "cmdline", "high", append([]string{args[0]}, fileArgs(args, ".py")...)...)
	case isAny(first, "node", "nodejs", "bun", "deno") || hasSuffixAny(args, ".js", ".mjs", ".cjs", ".ts"):
		return match("nodejs", "cmdline", "high", append([]string{args[0]}, fileArgs(args, ".js", ".mjs", ".cjs", ".ts")...)...)
	case isAny(first, "dotnet", "mono") || hasSuffixAny(args, ".dll"):
		return match("dotnet", "cmdline", "high", append([]string{args[0]}, fileArgs(args, ".dll")...)...)
	case isAny(first, "ruby") || hasSuffixAny(args, ".rb"):
		return match("ruby", "cmdline", "high", append([]string{args[0]}, fileArgs(args, ".rb")...)...)
	case isAny(first, "php") || hasSuffixAny(args, ".php"):
		return match("php", "cmdline", "high", append([]string{args[0]}, fileArgs(args, ".php")...)...)
	case isAny(first, "perl") || hasSuffixAny(args, ".pl", ".pm"):
		return match("perl", "cmdline", "high", append([]string{args[0]}, fileArgs(args, ".pl", ".pm")...)...)
	case isAny(first, "cargo", "rustc") || hasSuffixAny(args, ".rs"):
		return match("rust", "cmdline", "high", append([]string{args[0]}, fileArgs(args, ".rs")...)...)
	case first == "go" && len(args) > 1 && isAny(strings.ToLower(args[1]), "run", "test", "build", "tool"):
		return match("go", "cmdline", "high", args[0], "go "+args[1])
	}

	return unknownDetection()
}

func detectFromEnv(env map[string]string) Detection {
	if len(env) == 0 {
		return unknownDetection()
	}

	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, strings.ToUpper(k))
	}
	sort.Strings(keys)

	hasPrefix := func(prefixes ...string) bool {
		for _, key := range keys {
			for _, prefix := range prefixes {
				if strings.HasPrefix(key, prefix) {
					return true
				}
			}
		}
		return false
	}

	switch {
	case hasPrefix("PYTHONPATH", "VIRTUAL_ENV", "CONDA_PREFIX"):
		return match("python", "env", "medium", envEvidence(env, "PYTHONPATH", "VIRTUAL_ENV", "CONDA_PREFIX")...)
	case hasPrefix("NODE_OPTIONS", "NPM_", "YARN_"):
		return match("nodejs", "env", "medium", envEvidence(env, "NODE_OPTIONS", "NPM_", "YARN_")...)
	case hasPrefix("DOTNET_", "ASPNETCORE_", "COREHOST_"):
		return match("dotnet", "env", "medium", envEvidence(env, "DOTNET_", "ASPNETCORE_", "COREHOST_")...)
	case hasPrefix("GOPATH", "GOMOD", "GOMODCACHE", "GOCACHE"):
		return match("go", "env", "medium", envEvidence(env, "GOPATH", "GOMOD", "GOMODCACHE", "GOCACHE")...)
	case hasPrefix("GEM_HOME", "BUNDLE_"):
		return match("ruby", "env", "medium", envEvidence(env, "GEM_HOME", "BUNDLE_")...)
	case hasPrefix("PHPRC", "PHP_INI_SCAN_DIR"):
		return match("php", "env", "medium", envEvidence(env, "PHPRC", "PHP_INI_SCAN_DIR")...)
	case hasPrefix("PERL5LIB"):
		return match("perl", "env", "medium", envEvidence(env, "PERL5LIB")...)
	case hasPrefix("CARGO_HOME", "RUSTUP_HOME"):
		return match("rust", "env", "medium", envEvidence(env, "CARGO_HOME", "RUSTUP_HOME")...)
	}

	return unknownDetection()
}

func detectFromPaths(cwd, executable string) Detection {
	candidates := uniqueStrings([]string{
		cwd,
		filepath.Dir(executable),
		parentDir(cwd),
		parentDir(filepath.Dir(executable)),
	})

	for _, dir := range candidates {
		if dir == "" || dir == "." || dir == string(filepath.Separator) {
			continue
		}

		switch {
		case fileExists(filepath.Join(dir, "go.mod")) || fileExists(filepath.Join(dir, "go.sum")):
			return match("go", "cwd", "high", filepath.Join(dir, "go.mod"))
		case fileExists(filepath.Join(dir, "package.json")) || fileExists(filepath.Join(dir, "node_modules")):
			return match("nodejs", "cwd", "high", filepath.Join(dir, "package.json"))
		case fileExists(filepath.Join(dir, "pyproject.toml")) || fileExists(filepath.Join(dir, "requirements.txt")) || fileExists(filepath.Join(dir, "setup.py")):
			return match("python", "cwd", "high", firstExisting(filepath.Join(dir, "pyproject.toml"), filepath.Join(dir, "requirements.txt"), filepath.Join(dir, "setup.py")))
		case fileExists(filepath.Join(dir, "Cargo.toml")):
			return match("rust", "cwd", "high", filepath.Join(dir, "Cargo.toml"))
		case fileExists(filepath.Join(dir, "Gemfile")):
			return match("ruby", "cwd", "high", filepath.Join(dir, "Gemfile"))
		case fileExists(filepath.Join(dir, "composer.json")):
			return match("php", "cwd", "high", filepath.Join(dir, "composer.json"))
		case fileExists(filepath.Join(dir, "appsettings.json")) || fileExists(filepath.Join(dir, "web.config")) || hasAnyFileSuffix(dir, ".dll", ".deps.json", ".runtimeconfig.json"):
			return match("dotnet", "cwd", "high", dir)
		}
	}

	return unknownDetection()
}

func detectFromExecutable(executable, comm string, cmdline []string) Detection {
	base := strings.ToLower(filepath.Base(executable))
	comm = strings.ToLower(comm)

	switch {
	case isAny(base, "python", "python2", "python3") || strings.Contains(comm, "python"):
		return match("python", "exe", "medium", executable)
	case isAny(base, "node", "nodejs", "bun", "deno") || strings.Contains(comm, "node"):
		return match("nodejs", "exe", "medium", executable)
	case isAny(base, "dotnet", "mono") || strings.Contains(comm, "dotnet"):
		return match("dotnet", "exe", "medium", executable)
	case isAny(base, "ruby") || strings.Contains(comm, "ruby"):
		return match("ruby", "exe", "medium", executable)
	case isAny(base, "php") || strings.Contains(comm, "php"):
		return match("php", "exe", "medium", executable)
	case isAny(base, "perl") || strings.Contains(comm, "perl"):
		return match("perl", "exe", "medium", executable)
	case isAny(base, "cargo", "rustc") || strings.Contains(comm, "rust"):
		return match("rust", "exe", "medium", executable)
	}

	if len(cmdline) > 0 {
		first := strings.ToLower(filepath.Base(cmdline[0]))
		if isAny(first, "python", "python2", "python3") {
			return match("python", "exe", "medium", cmdline[0])
		}
	}

	return unknownDetection()
}

func match(language, method, confidence string, evidence ...string) Detection {
	return Detection{
		Language:   language,
		Method:     method,
		Confidence: confidence,
		Evidence:   compactEvidence(evidence),
	}
}

func unknownDetection() Detection {
	return Detection{
		Language:   "unknown",
		Method:     "heuristic",
		Confidence: "low",
		Evidence:   nil,
	}
}

func compactEvidence(items []string) []string {
	seen := make(map[string]struct{}, len(items))
	out := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}

func envEvidence(env map[string]string, patterns ...string) []string {
	out := make([]string, 0, len(patterns))
	for key := range env {
		upper := strings.ToUpper(key)
		for _, pattern := range patterns {
			if strings.HasPrefix(upper, pattern) {
				out = append(out, key)
				break
			}
		}
	}
	sort.Strings(out)
	return out
}

func fileArgs(args []string, suffixes ...string) []string {
	out := make([]string, 0)
	for _, arg := range args {
		lower := strings.ToLower(arg)
		for _, suffix := range suffixes {
			if strings.HasSuffix(lower, suffix) {
				out = append(out, arg)
				break
			}
		}
	}
	return out
}

func hasSuffixAny(args []string, suffixes ...string) bool {
	for _, arg := range args {
		lower := strings.ToLower(arg)
		for _, suffix := range suffixes {
			if strings.HasSuffix(lower, suffix) {
				return true
			}
		}
	}
	return false
}

func isAny(value string, candidates ...string) bool {
	for _, candidate := range candidates {
		if value == candidate {
			return true
		}
	}
	return false
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func parentDir(path string) string {
	if path == "" || path == "." || path == string(filepath.Separator) {
		return ""
	}
	return filepath.Dir(path)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func firstExisting(paths ...string) string {
	for _, path := range paths {
		if fileExists(path) {
			return path
		}
	}
	if len(paths) > 0 {
		return paths[0]
	}
	return ""
}

func hasAnyFileSuffix(dir string, suffixes ...string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := strings.ToLower(entry.Name())
		for _, suffix := range suffixes {
			if strings.HasSuffix(name, suffix) {
				return true
			}
		}
	}
	return false
}
