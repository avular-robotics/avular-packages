package adapters

import (
	"sort"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
)

func TestParseAptPackages(t *testing.T) {
	content := strings.Join([]string{
		"Package: libfoo",
		"Version: 1.0.0",
		"Depends: libc6 (>= 2.31), libbar | libbaz",
		"Pre-Depends: dpkg (>= 1.19)",
		"Provides: foo-virtual",
		"",
		"Package: libfoo",
		"Version: 1.1.0",
		"",
		"Package: libbar",
		"Version: 2.0.0",
		"",
	}, "\n")
	index, err := parseAptPackages(strings.NewReader(content))
	require.NoError(t, err)
	barVersions := index["libbar"]
	if _, ok := barVersions["2.0.0"]; !ok {
		t.Fatalf("missing libbar version")
	}
	fooVersions := index["libfoo"]
	if _, ok := fooVersions["1.0.0"]; !ok {
		t.Fatalf("missing libfoo version 1.0.0")
	}
	if _, ok := fooVersions["1.1.0"]; !ok {
		t.Fatalf("missing libfoo version 1.1.0")
	}
	deps := fooVersions["1.0.0"].Depends
	if diff := cmp.Diff([]string{"libc6 (>= 2.31)", "libbar | libbaz"}, deps); diff != "" {
		t.Fatalf("unexpected depends (-want +got):\n%s", diff)
	}
	pre := fooVersions["1.0.0"].PreDepends
	if diff := cmp.Diff([]string{"dpkg (>= 1.19)"}, pre); diff != "" {
		t.Fatalf("unexpected pre-depends (-want +got):\n%s", diff)
	}
	provides := fooVersions["1.0.0"].Provides
	if diff := cmp.Diff([]string{"foo-virtual"}, provides); diff != "" {
		t.Fatalf("unexpected provides (-want +got):\n%s", diff)
	}
}

func TestParsePipSimpleNames(t *testing.T) {
	tests := []struct {
		name string
		html string
		want []string
	}{
		{
			name: "basic",
			html: `<a href="Foo/">Foo</a><a href="requests/">requests</a>`,
			want: []string{"foo", "requests"},
		},
		{
			name: "dedupe and normalize",
			html: `<a href="Django/">Django</a><a href="django/">django</a><a href="my_pkg/">my_pkg</a>`,
			want: []string{"django", "my-pkg"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			names := parsePipSimpleNames(tt.html)
			if diff := cmp.Diff(tt.want, names); diff != "" {
				t.Fatalf("unexpected names (-want +got):\n%s", diff)
			}
		})
	}
}

func TestParsePipVersionsFromSimple(t *testing.T) {
	tests := []struct {
		name string
		html string
		want []string
	}{
		{
			name: "wheel and sdist",
			html: `<a href="requests-2.31.0-py3-none-any.whl">whl</a>` +
				`<a href="requests-2.32.0.tar.gz">sdist</a>`,
			want: []string{"2.31.0", "2.32.0"},
		},
		{
			name: "filters invalid filenames",
			html: `<a href="demo.whl">bad</a><a href="demo-1.0.0.tar.gz">ok</a>`,
			want: []string{"1.0.0"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			versions := parsePipVersionsFromSimple(tt.html)
			sort.Strings(versions)
			if diff := cmp.Diff(tt.want, versions); diff != "" {
				t.Fatalf("unexpected versions (-want +got):\n%s", diff)
			}
		})
	}
}

func TestParsePipVersionFromFilename(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     string
	}{
		{
			name:     "wheel",
			filename: "demo-1.2.3-py3-none-any.whl",
			want:     "1.2.3",
		},
		{
			name:     "sdist",
			filename: "demo-4.5.6.tar.gz",
			want:     "4.5.6",
		},
		{
			name:     "missing version",
			filename: "demo.whl",
			want:     "",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if diff := cmp.Diff(tt.want, parsePipVersionFromFilename(tt.filename)); diff != "" {
				t.Fatalf("unexpected version (-want +got):\n%s", diff)
			}
		})
	}
}
