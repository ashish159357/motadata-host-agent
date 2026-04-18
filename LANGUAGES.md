# Language Detection Coverage

> Maintained automatically by `/detect-language <lang>` after every successful run.
> Read by `/list-languages` to produce the coverage report.
> Do not edit by hand — agent writes overwrite this file.

## Coverage Table

| Language | E (environ) | C (cmdline) | F (files) | M (maps) | X (exepath) | Last updated |
|----------|:-----------:|:-----------:|:---------:|:--------:|:-----------:|--------------|
| go       | ✅ | ✅ | ✅ | ✗ | ✅ | 2026-04-17 |
| java     | ✅ | ✅ | ✅ | ✅ | ✅ | 2026-04-17 |
| python   | ✅ | ✅ | ✅ | ✅ | ✅ | — |
| nodejs   | ✅ | ✅ | ✅ | ✅ | ✅ | — |
| dotnet   | ✅ | ✅ | ✅ | ✅ | ✅ | — |
| ruby     | ✅ | ✅ | ✅ | ✅ | ✅ | — |
| php      | ✅ | ✅ | ✅ | ✅ | ✅ | — |
| perl     | ✗ | ✅ | ✗ | ✗ | ✗ | — |
| rust     | ✅ | ✗ | ✅ | ✗ | ✅ | — |
| cpp      | ✗ | ✗ | ✅ | ✅ | ✗ | — |

**Stages key:** E=environ · C=cmdline · F=files · M=maps · X=exepath

## Environ Signals

| Language | Signals |
|----------|---------|
| go       | GOMEMLIMIT, GOGC, GOMAXPROCS, GOPATH, GOFLAGS, GONOSUMDB, GONOSUMCHECK |
| java     | JAVA_TOOL_OPTIONS, JAVA_OPTS, CATALINA_HOME, CATALINA_BASE, JVM_OPTS, JAVA_HOME, JAVA_VERSION, CLASSPATH, JDK_JAVA_OPTIONS |
| python   | PYTHONPATH, PYTHONHOME, VIRTUAL_ENV, CONDA_* |
| nodejs   | NODE_ENV, NODE_OPTIONS, NODE_PATH, NPM_CONFIG_PREFIX |
| dotnet   | ASPNETCORE_*, DOTNET_* |
| ruby     | RAILS_ENV, RACK_ENV, BUNDLE_PATH |
| php      | PHP_INI_SCAN_DIR, PHPRC |
| perl     | — |
| rust     | CARGO_HOME, RUSTUP_HOME |
| cpp      | — |

## CmdLine Signals

| Language | Signals |
|----------|---------|
| go       | `go` binary; `.go` arg |
| java     | java, javaw; `-jar` flag; `.jar` arg |
| python   | python, python2, python3; `.py` arg |
| nodejs   | node, nodejs, npm, npx, yarn, pnpm; `.js`, `.mjs` args |
| dotnet   | dotnet |
| ruby     | ruby, bundle; `.rb` arg |
| php      | php, php-fpm; `.php` arg |
| perl     | perl, perl5 |
| rust     | — |
| cpp      | — |

## File Markers

| Language | Markers |
|----------|---------|
| go       | go.mod, go.sum, go.work, go.work.sum, *.go |
| java     | pom.xml, build.gradle, build.gradle.kts, settings.gradle, gradlew, *.jar, *.war, *.ear |
| python   | requirements.txt, setup.py, pyproject.toml, *.py |
| nodejs   | package.json, node_modules |
| dotnet   | appsettings.json, web.config, *.pdb, *.deps.json |
| ruby     | Gemfile, Gemfile.lock, .ruby-version, *.rb, *.gemspec |
| php      | composer.json, composer.lock, *.php |
| perl     | — |
| rust     | Cargo.toml, Cargo.lock |
| cpp      | CMakeLists.txt (+ *.cpp/cc/cxx/hpp), meson.build, Makefile |

## Maps Signals

| Language | Signals |
|----------|---------|
| go       | — (statically compiled) |
| java     | libjvm.so, libjava.so |
| python   | libpython |
| nodejs   | libnode.so |
| dotnet   | libcoreclr, libmono |
| ruby     | libruby |
| php      | libphp |
| perl     | — |
| rust     | — (statically compiled) |
| cpp      | libstdc++.so, libgcc_s.so |

## ExePath Signals

| Language | Signals |
|----------|---------|
| go       | /go/bin/, golang, /usr/local/go/ |
| java     | jdk, jre, openjdk, java- |
| python   | python, pyenv |
| nodejs   | nodejs, /node/bin/ |
| dotnet   | dotnet |
| ruby     | ruby, rbenv, /rvm/ |
| php      | /php/, phpenv |
| perl     | — |
| rust     | rustup, /.cargo/ |
| cpp      | — |

## Change Log

| Date | Command | Language | Signals added | Tests added |
|------|---------|----------|---------------|-------------|
| 2026-04-17 | /detect-language java | java | environ×3, cmdline×1, files×2, exepath×1 | 8 |
| 2026-04-17 | /detect-language go | go | environ×4, cmdline×1, files×3, exepath×1 | 7 |
