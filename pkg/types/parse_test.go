package types

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

const (
	invalidFile        = "testdata/invalid.yaml"
	testFile           = "testdata/bumpfile.yaml"
	missingNameFile    = "testdata/missingname.yaml"
	missingVersionFile = "testdata/missingversion.yaml"
)

func TestParse(t *testing.T) {
	testCases := []struct {
		name     string
		bumpFile string
		want     map[string]*Package
		wantErr  string
	}{{
		name:     "no file",
		bumpFile: "",
		wantErr:  "no filename specified",
	}, {
		name:     "file not found",
		bumpFile: "testdata/missing",
		wantErr:  "failed reading file",
	}, {
		name:     "missing version",
		bumpFile: missingVersionFile,
		wantErr:  "missing version",
	}, {
		name:     "missing name",
		bumpFile: missingNameFile,
		wantErr:  "missing name",
	}, {
		name:     "invalid file",
		bumpFile: invalidFile,
		wantErr:  "unmarshaling file",
	}, {
		name:     "file",
		bumpFile: testFile,
		want: map[string]*Package{"name-1": {
			Name:    "name-1",
			Version: "version-1",
			Index:   0,
		},
			"name-2": {
				Name:    "name-2",
				Version: "version-2",
				Index:   1,
			},
			"name-3": {
				Name:    "name-3",
				Version: "version-3",
				Index:   2,
			}},
	}}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseFile(tc.bumpFile)
			if err != nil && (tc.wantErr == "") {
				t.Errorf("%s: ParseFile(%ss) = %v)", tc.name, tc.bumpFile, err)
			}
			if err != nil && tc.wantErr != "" {
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Errorf("%s: ParseFile(%s) = %v, want %v", tc.name, tc.bumpFile, err, tc.wantErr)
				}
				return
			}
			// We don't care about the order of the patches
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("%s: ParseFIle(%s) (-got +want)\n%s", tc.name, tc.bumpFile, diff)
			}
		})
	}
}
