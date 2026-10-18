package launchwrapper

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const windowsOS = "windows"

func resolveExecutable(name string, ctx Context) (string, error) {
	if filepath.IsAbs(name) {
		return executableFile(name, ctx.Env)
	}
	if strings.ContainsAny(name, `/\`) {
		return executableFile(filepath.Join(ctx.WorkDir, name), ctx.Env)
	}
	for _, dir := range filepath.SplitList(envValue(ctx.Env, "PATH")) {
		// Names never search the current directory implicitly, including relative PATH entries.
		if !filepath.IsAbs(dir) {
			continue
		}
		if path, err := executableFile(filepath.Join(dir, name), ctx.Env); err == nil {
			return path, nil
		}
	}
	return "", ErrUnavailable
}

func envValue(env []string, key string) string {
	for i := len(env) - 1; i >= 0; i-- {
		name, value, found := strings.Cut(env[i], "=")
		if found && (name == key || runtime.GOOS == windowsOS && strings.EqualFold(name, key)) {
			return value
		}
	}
	return ""
}

func executableFile(path string, env []string) (string, error) {
	candidates := []string{path}
	if runtime.GOOS == windowsOS {
		extensions := envValue(env, "PATHEXT")
		if extensions == "" {
			extensions = ".COM;.EXE;.BAT;.CMD"
		}
		for _, ext := range strings.Split(extensions, ";") {
			if strings.HasPrefix(ext, ".") && !strings.ContainsAny(ext, `/\`) {
				candidates = append(candidates, path+ext)
			}
		}
	}
	for _, candidate := range candidates {
		info, err := os.Stat(candidate)
		if err != nil || !info.Mode().IsRegular() {
			continue
		}
		if runtime.GOOS != windowsOS && info.Mode().Perm()&0o111 == 0 {
			continue
		}
		return candidate, nil
	}
	return "", ErrUnavailable
}
