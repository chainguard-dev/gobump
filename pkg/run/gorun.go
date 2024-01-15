package run

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	versionutil "k8s.io/apimachinery/pkg/util/version"
)

func GoModTidy(modroot, goVersion string) (string, error) {
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

		log.Printf("Running go mod tidy with go version '%s' ...\n", goVersion)
	}

	cmd := exec.Command("go", "mod", "tidy", "-go", goVersion)
	cmd.Dir = modroot
	if bytes, err := cmd.CombinedOutput(); err != nil {
		return strings.TrimSpace(string(bytes)), err
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
