package update

import (
	"os/exec"
	"testing"

	"github.com/chainguard-dev/gobump/pkg/types"
)

func TestUpdate(t *testing.T) {
	testCases := []struct {
		name        string
		pkgVersions map[string]*types.Package
		want        map[string]string
	}{
		{
			name: "standard update",
			pkgVersions: map[string]*types.Package{
				"github.com/google/uuid": {
					Name:    "github.com/google/uuid",
					Version: "v1.4.0",
				},
			},
			want: map[string]string{
				"github.com/google/uuid": "v1.4.0",
			},
		},
		{
			name: "replace",
			pkgVersions: map[string]*types.Package{
				"k8s.io/client-go": {
					OldName: "k8s.io/client-go",
					Name:    "k8s.io/client-go",
					Version: "v0.28.0",
				},
			},
			want: map[string]string{
				"k8s.io/client-go": "v0.28.0",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpdir := t.TempDir()
			copyFile(t, "testdata/aws-efs-csi-driver/go.mod", tmpdir)

			modFile, err := DoUpdate(tc.pkgVersions, &types.Config{Modroot: tmpdir, Tidy: false, GoVersion: ""})
			if err != nil {
				t.Fatal(err)
			}
			for pkg, want := range tc.want {
				if got := getVersion(modFile, pkg); got != want {
					t.Errorf("expected %s, got %s", want, got)
				}
			}
		})
	}
}

func TestGoModTidy(t *testing.T) {
	testCases := []struct {
		name        string
		pkgVersions map[string]*types.Package
		want        map[string]string
	}{
		{
			name: "standard update",
			pkgVersions: map[string]*types.Package{
				"github.com/sirupsen/logrus": {
					Name:    "github.com/sirupsen/logrus",
					Version: "v1.9.0",
				},
			},
			want: map[string]string{
				"github.com/sirupsen/logrus": "v1.9.0",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpdir := t.TempDir()
			copyFile(t, "testdata/hello/go.mod", tmpdir)
			copyFile(t, "testdata/hello/go.sum", tmpdir)
			copyFile(t, "testdata/hello/main.go", tmpdir)

			modFile, err := DoUpdate(tc.pkgVersions, &types.Config{Modroot: tmpdir, Tidy: false, GoVersion: ""})
			if err != nil {
				t.Fatal(err)
			}
			for pkg, want := range tc.want {
				if got := getVersion(modFile, pkg); got != want {
					t.Errorf("expected %s, got %s", want, got)
				}
			}
		})
	}
}

func TestReplaceAndRequire(t *testing.T) {
	testCases := []struct {
		name        string
		pkgVersions map[string]*types.Package
		want        map[string]string
	}{
		{
			name: "standard update",
			pkgVersions: map[string]*types.Package{
				"github.com/sirupsen/logrus": {
					Name:    "github.com/sirupsen/logrus",
					Version: "v1.9.0",
				},
			},
			want: map[string]string{
				"github.com/sirupsen/logrus": "v1.9.0",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpdir := t.TempDir()
			copyFile(t, "testdata/bye/go.mod", tmpdir)
			copyFile(t, "testdata/bye/go.sum", tmpdir)
			copyFile(t, "testdata/bye/main.go", tmpdir)

			modFile, err := DoUpdate(tc.pkgVersions, &types.Config{Modroot: tmpdir, Tidy: false, GoVersion: ""})
			if err != nil {
				t.Fatal(err)
			}
			for pkg, want := range tc.want {
				if got := getVersion(modFile, pkg); got != want {
					t.Errorf("expected %s, got %s", want, got)
				}
			}
		})
	}
}

func TestUpdateError(t *testing.T) {
	testCases := []struct {
		name        string
		pkgVersions map[string]*types.Package
	}{
		{
			name: "no downgrade",
			pkgVersions: map[string]*types.Package{
				"github.com/google/uuid": {
					Name:    "github.com/google/uuid",
					Version: "v1.0.0",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpdir := t.TempDir()
			copyFile(t, "testdata/aws-efs-csi-driver/go.mod", tmpdir)

			_, err := DoUpdate(tc.pkgVersions, &types.Config{Modroot: tmpdir, Tidy: false, GoVersion: ""})
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}

func TestReplaces(t *testing.T) {
	tmpdir := t.TempDir()
	copyFile(t, "testdata/aws-efs-csi-driver/go.mod", tmpdir)

	replaces := map[string]*types.Package{
		"github.com/google/gofuzz": {
			OldName: "github.com/google/gofuzz",
			Name:    "github.com/fakefuzz",
			Version: "v1.2.3",
			Replace: true,
		}}

	modFile, err := DoUpdate(replaces, &types.Config{Modroot: tmpdir, Tidy: false, GoVersion: ""})
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range modFile.Replace {
		if r.Old.Path == "github.com/google/gofuzz" {
			if r.New.Path != "github.com/fakefuzz" {
				t.Errorf("expected replace of github.com/google/gofuzz with github.com/fakefuzz, got %s", r.New.Path)
			}
			if r.Old.Path != "github.com/google/gofuzz" {
				t.Errorf("expected replace of github.com/google/gofuzz, got %s", r.Old.Path)
			}
			if r.New.Version != "v1.2.3" {
				t.Errorf("expected replace of github.com/google/gofuzz with v1.2.3, got %s", r.New.Version)
			}
			break
		}
	}
}

func TestCommit(t *testing.T) {
	// We use github.com/NVIDIA/go-nvml v0.11.7-0 in our go.mod
	// That corresponds to 53c34bc04d66e9209eff8654bc70563cf380e214
	pkg := "github.com/NVIDIA/go-nvml"

	// An older commit is c3a16a2b07cf2251cbedb76fa68c9292b22bfa06
	olderCommit := "c3a16a2b07cf2251cbedb76fa68c9292b22bfa06"
	olderVersion := "v0.11.6-0"
	// A newer commit is 95ef6acc3271a9894fd02c1071edef1d88527e20
	newerCommit := "95ef6acc3271a9894fd02c1071edef1d"
	newerVersion := "v0.12.0-1"

	testCases := []struct {
		name    string
		version string
		want    map[string]string
	}{
		{
			name:    "pin to older",
			version: olderCommit,
			want: map[string]string{
				pkg: olderVersion,
			},
		},
		{
			name:    "pin to newer",
			version: newerCommit,
			want: map[string]string{
				pkg: newerVersion,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpdir := t.TempDir()
			copyFile(t, "testdata/aws-efs-csi-driver/go.mod", tmpdir)

			pkgVersions := map[string]*types.Package{
				pkg: {
					Name:    pkg,
					Version: tc.version,
				},
			}
			modFile, err := DoUpdate(pkgVersions, &types.Config{Modroot: tmpdir, Tidy: false, GoVersion: "1.21"})
			if err != nil {
				t.Fatal(err)
			}
			for pkg, want := range tc.want {
				if got := getVersion(modFile, pkg); got != want {
					t.Errorf("expected %s, got %s", want, got)
				}
			}
		})
	}

}

func copyFile(t *testing.T, src, dst string) {
	t.Helper()
	_, err := exec.Command("cp", "-r", src, dst).Output()
	if err != nil {
		t.Fatal(err)
	}
}
