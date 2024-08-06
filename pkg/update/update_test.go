package update

import (
	"os/exec"
	"reflect"
	"strings"
	"testing"

	"github.com/chainguard-dev/gobump/pkg/types"
)

// maybeParseFile will parse the file if the filename is not empty
// On failure, fatals to simplify the error handling in tests.
func maybeParseFile(t *testing.T, fileName string, packages map[string]*types.Package) map[string]*types.Package {
	if fileName != "" {
		ret, err := types.ParseFile(fileName)
		if err != nil {
			t.Fatalf("Failed to parse file %q: %v", fileName, err)
		}
		return ret
	}
	return packages
}

func TestUpdate(t *testing.T) {
	testCases := []struct {
		name        string
		pkgVersions map[string]*types.Package
		fileName    string
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
			name:     "standard update - from file",
			fileName: "testdata/standardUpdate.yaml",
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
		{
			name:     "replace - from file",
			fileName: "testdata/standardReplace.yaml",
			want: map[string]string{
				"k8s.io/client-go": "v0.28.0",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpdir := t.TempDir()
			copyFile(t, "testdata/aws-efs-csi-driver/go.mod", tmpdir)
			pkgVersions := maybeParseFile(t, tc.fileName, tc.pkgVersions)
			modFile, err := DoUpdate(pkgVersions, &types.Config{Modroot: tmpdir, Tidy: false, GoVersion: ""})
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

func TestUpdateInOrder(t *testing.T) {
	testCases := []struct {
		name        string
		pkgVersions map[string]*types.Package
		fileName    string
		want        []string
	}{
		{
			name: "standard update",
			pkgVersions: map[string]*types.Package{
				"github.com/google/uuid": {
					Name:    "github.com/google/uuid",
					Version: "v1.4.0",
					Index:   0,
				},
				"k8s.io/api": {
					OldName: "k8s.io/api",
					Name:    "k8s.io/api",
					Version: "v0.28.0",
					Index:   2,
				},
				"k8s.io/client-go": {
					OldName: "k8s.io/client-go",
					Name:    "k8s.io/client-go",
					Version: "v0.28.0",
					Index:   1,
				},
			},
			want: []string{
				"github.com/google/uuid",
				"k8s.io/api",
				"k8s.io/client-go",
			},
		},
		{
			name:     "standard update - file",
			fileName: "testdata/inorder.yaml",
			want: []string{
				"github.com/google/uuid",
				"k8s.io/api",
				"k8s.io/client-go",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpdir := t.TempDir()
			copyFile(t, "testdata/aws-efs-csi-driver/go.mod", tmpdir)

			pkgVersions := maybeParseFile(t, tc.fileName, tc.pkgVersions)
			got := orderPkgVersionsMap(pkgVersions)
			if len(got) != len(tc.want) || reflect.DeepEqual(got, tc.want) {
				t.Errorf("expected %s, got %s", tc.want, got)
			}
		})
	}
}

func TestGoModTidy(t *testing.T) {
	testCases := []struct {
		name        string
		pkgVersions map[string]*types.Package
		fileName    string
		want        map[string]string
		wantErr     bool
		errMsg      string
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
		{
			name:     "standard update - file",
			fileName: "testdata/logrus.yaml",
			want: map[string]string{
				"github.com/sirupsen/logrus": "v1.9.0",
			},
		}, {
			name: "error when bumping main module",
			pkgVersions: map[string]*types.Package{
				"github.com/puerco/hello": {
					Name:    "github.com/puerco/hello",
					Version: "v1.9.0",
				},
			},
			wantErr: true,
			errMsg:  "bumping the main module is not allowed 'github.com/puerco/hello'",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpdir := t.TempDir()
			copyFile(t, "testdata/hello/go.mod", tmpdir)
			copyFile(t, "testdata/hello/go.sum", tmpdir)
			copyFile(t, "testdata/hello/main.go", tmpdir)

			pkgVersions := maybeParseFile(t, tc.fileName, tc.pkgVersions)
			modFile, err := DoUpdate(pkgVersions, &types.Config{Modroot: tmpdir, Tidy: false, GoVersion: ""})
			if (err != nil) != tc.wantErr {
				t.Errorf("DoUpdate() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if tc.wantErr && err.Error() != tc.errMsg {
				t.Errorf("expected err message %s, got %s", tc.errMsg, err.Error())
			}
			for pkg, want := range tc.want {
				if got := getVersion(modFile, pkg); got != want {
					t.Errorf("expected %s, got %s", want, got)
				}
			}
		})
	}
}

func TestGoModTidySkipInitial(t *testing.T) {
	testCases := []struct {
		name            string
		pkgVersions     map[string]*types.Package
		tidySkipInitial bool
		wantError       bool
		want            map[string]string
		errMsgContains  string
	}{
		{
			name: "do not skip initial tidy",
			pkgVersions: map[string]*types.Package{
				"github.com/coreos/etcd": {
					Name:    "github.com/coreos/etcd",
					Version: "v3.3.15",
				},
				"google.golang.org/grpc": {
					Name:    "google.golang.org/grpc",
					OldName: "google.golang.org/grpc",
					Version: "v1.29.0",
					Replace: true,
				},
			},
			tidySkipInitial: false,
			wantError:       true,
			errMsgContains:  "ambiguous import",
		},
		{
			name: "skip initial tidy",
			pkgVersions: map[string]*types.Package{
				"github.com/coreos/etcd": {
					Name:    "github.com/coreos/etcd",
					Version: "v3.3.15",
				},
				"google.golang.org/grpc": {
					Name:    "google.golang.org/grpc",
					OldName: "google.golang.org/grpc",
					Version: "v1.29.0",
					Replace: true,
				},
			},
			tidySkipInitial: true,
			wantError:       true,
			errMsgContains:  "Please remove the package or add it to the list of 'replaces'",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpdir := t.TempDir()
			copyFile(t, "testdata/confd/go.mod", tmpdir)
			copyFile(t, "testdata/confd/go.sum", tmpdir)

			modFile, err := DoUpdate(tc.pkgVersions, &types.Config{Modroot: tmpdir, Tidy: true, GoVersion: "1.19", TidySkipInitial: tc.tidySkipInitial})
			if (err != nil) != tc.wantError {
				t.Errorf("DoUpdate() error = %v, wantErr %v", err, tc.wantError)
				return
			}
			if tc.wantError && !strings.Contains(err.Error(), tc.errMsgContains) {
				t.Errorf("expected err message not contains %s, got %s", tc.errMsgContains, err.Error())
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
		fileName    string
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
		{
			name:     "standard update - file",
			fileName: "testdata/logrus.yaml",
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

			pkgVersions := maybeParseFile(t, tc.fileName, tc.pkgVersions)
			modFile, err := DoUpdate(pkgVersions, &types.Config{Modroot: tmpdir, Tidy: false, GoVersion: ""})
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
		fileName    string
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
		{
			name:     "no downgrade - from file",
			fileName: "testdata/nodowngrade.yaml",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpdir := t.TempDir()
			copyFile(t, "testdata/aws-efs-csi-driver/go.mod", tmpdir)

			pkgVersions := maybeParseFile(t, tc.fileName, tc.pkgVersions)
			_, err := DoUpdate(pkgVersions, &types.Config{Modroot: tmpdir, Tidy: false, GoVersion: ""})
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}

func TestReplaces(t *testing.T) {
	testCases := []struct {
		name     string
		replaces map[string]*types.Package
		fileName string
	}{
		{
			name: "replace",
			replaces: map[string]*types.Package{
				"github.com/google/gofuzz": {
					OldName: "github.com/google/gofuzz",
					Name:    "github.com/fakefuzz",
					Version: "v1.2.3",
					Replace: true,
				}},
		},
		{
			name:     "replace - from file",
			fileName: "testdata/replaces.yaml",
		},
	}

	for _, tc := range testCases {
		tmpdir := t.TempDir()
		copyFile(t, "testdata/aws-efs-csi-driver/go.mod", tmpdir)

		replaces := maybeParseFile(t, tc.fileName, tc.replaces)
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
		name        string
		pkgVersions map[string]*types.Package
		fileName    string
		want        map[string]string
	}{
		{
			name: "pin to older",
			pkgVersions: map[string]*types.Package{
				pkg: {Name: pkg, Version: olderCommit},
			},
			want: map[string]string{
				pkg: olderVersion,
			},
		},
		{
			name:     "pin to older - file",
			fileName: "testdata/older.yaml",
			want: map[string]string{
				pkg: olderVersion,
			},
		},
		{
			name: "pin to newer",
			pkgVersions: map[string]*types.Package{
				pkg: {Name: pkg, Version: newerCommit},
			},
			want: map[string]string{
				pkg: newerVersion,
			},
		},
		{
			name:     "pin to newer - file",
			fileName: "testdata/newer.yaml",
			want: map[string]string{
				pkg: newerVersion,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpdir := t.TempDir()
			copyFile(t, "testdata/aws-efs-csi-driver/go.mod", tmpdir)

			pkgVersions := maybeParseFile(t, tc.fileName, tc.pkgVersions)
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

func TestParseGoVersionString(t *testing.T) {
	tests := []struct {
		name          string
		versionOutput string
		want          string
		wantErr       bool
	}{
		{"valid version 1.15.2", "go version go1.15.2 linux/amd64", "1.15.2", false},
		{"valid version 1.21.6", "go version go1.21.6 darwin/arm64", "1.21.6", false},
		{"go not found", "sh: go: not found", "", true},
		{"unexpected format", "unexpected format string", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseGoVersionString(tt.versionOutput)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseGoVersionString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseGoVersionString() = %v, want %v", got, tt.want)
			}
		})
	}
}
