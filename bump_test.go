package main

import (
	"os/exec"
	"testing"
)

func TestUpdate(t *testing.T) {
	testCases := []struct {
		name        string
		pkgVersions []pkgVersion
		want        map[string]string
	}{
		{
			name: "standard update",
			pkgVersions: []pkgVersion{
				{
					Name:    "github.com/google/uuid",
					Version: "v1.4.0",
				},
			},
			want: map[string]string{
				"github.com/google/uuid": "v1.4.0",
			},
		},
		{
			name: "no downgrade",
			pkgVersions: []pkgVersion{
				{
					Name:    "github.com/google/uuid",
					Version: "v1.0.0",
				},
			},
			want: map[string]string{
				"github.com/google/uuid": "v1.3.1",
			},
		},
		{
			name: "replace",
			pkgVersions: []pkgVersion{
				{
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

			modFile, err := doUpdate(tc.pkgVersions, tmpdir)
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
