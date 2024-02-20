package run

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	versionutil "k8s.io/apimachinery/pkg/util/version"
)

func GoModTidy(modroot, goVersion, compat string) (string, error) {
	if goVersion == "" {
		cmd := exec.Command("go", "env", "GOVERSION")
		cmd.Stderr = os.Stderr
		out, err := cmd.Output()
		if err != nil {
			return "", fmt.Errorf("%v: %w", cmd, err)
		}
		goVersion = strings.TrimPrefix(strings.TrimSpace(string(out)), "go")

		v := versionutil.MustParseGeneric(goVersion)
		goVersion = fmt.Sprintf("%d.%d", v.Major(), v.Minor())
	}

	log.Printf("Running go mod tidy with go version '%s' ...\n", goVersion)
	args := []string{"mod", "tidy", "-go", goVersion}
	if compat != "" {
		log.Printf("Running go mod tidy with compat '%s' ...\n", compat)
		args = append(args, "-compat", compat)
	}

	cmd := exec.Command("go", args...)
	cmd.Dir = modroot
	if bytes, err := cmd.CombinedOutput(); err != nil {
		return strings.TrimSpace(string(bytes)), err
	}
	return "", nil
}

func findWorkspaceFile(dir string) (root string) {
	dir = filepath.Clean(dir)
	// Look for enclosing go.mod.
	for {
		f := filepath.Join(dir, "go.work")
		if fi, err := os.Stat(f); err == nil && !fi.IsDir() {
			return f
		}
		d := filepath.Dir(dir)
		if d == dir {
			break
		}
		dir = d
	}
	return ""
}

func findGoWork(modroot string) string {
	switch gowork := os.Getenv("GOWORK"); gowork {
	case "off":
		return ""
	case "", "auto":
		return findWorkspaceFile(modroot)
	default:
		return gowork
	}
}

func GoVendor(dir string) (string, error) {
	if findGoWork(dir) == "" {
		log.Print("Running go mod vendor...")
		cmd := exec.Command("go", "mod", "vendor")
		if bytes, err := cmd.CombinedOutput(); err != nil {
			return strings.TrimSpace(string(bytes)), err
		}
	} else {
		log.Print("Running go work vendor...")
		cmd := exec.Command("go", "work", "vendor")
		if bytes, err := cmd.CombinedOutput(); err != nil {
			return strings.TrimSpace(string(bytes)), err
		}
	}

	return "", nil
}

func GoGetModule(name, version, modroot string) (string, error) {
	cmd := exec.Command("go", "get", fmt.Sprintf("%s@%s", name, version)) //nolint:gosec
	cmd.Dir = modroot
	if bytes, err := cmd.CombinedOutput(); err != nil {
		return strings.TrimSpace(string(bytes)), err
	}
	return "", nil
}

func GoModEditReplaceModule(nameOld, nameNew, version, modroot string) (string, error) {
	cmd := exec.Command("go", "mod", "edit", "-dropreplace", nameOld) //nolint:gosec
	cmd.Dir = modroot
	if bytes, err := cmd.CombinedOutput(); err != nil {
		return strings.TrimSpace(string(bytes)), fmt.Errorf("Error running go command to drop replace modules: %w", err)
	}

	cmd = exec.Command("go", "mod", "edit", "-replace", fmt.Sprintf("%s=%s@%s", nameOld, nameNew, version)) //nolint:gosec
	cmd.Dir = modroot
	if bytes, err := cmd.CombinedOutput(); err != nil {
		return strings.TrimSpace(string(bytes)), fmt.Errorf("Error running go command to replace modules: %w", err)
	}
	return "", nil
}

func GoModEditDropRequireModule(name, modroot string) (string, error) {
	cmd := exec.Command("go", "mod", "edit", "-droprequire", name) //nolint:gosec
	cmd.Dir = modroot
	if bytes, err := cmd.CombinedOutput(); err != nil {
		return strings.TrimSpace(string(bytes)), err
	}

	return "", nil
}

func GoModEditRequireModule(name, version, modroot string) (string, error) {
	if bytes, err := GoModEditDropRequireModule(name, modroot); err != nil {
		return strings.TrimSpace(string(bytes)), err
	}

	cmd := exec.Command("go", "mod", "edit", "-require", fmt.Sprintf("%s@%s", name, version)) //nolint:gosec
	cmd.Dir = modroot
	if bytes, err := cmd.CombinedOutput(); err != nil {
		return strings.TrimSpace(string(bytes)), err
	}
	return "", nil
}
