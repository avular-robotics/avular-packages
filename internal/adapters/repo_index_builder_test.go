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
	expectedBar := []string{"2.0.0"}
	if diff := cmp.Diff(expectedBar, index["libbar"]); diff != "" {
		t.Fatalf("unexpected libbar versions (-want +got):\n%s", diff)
	}
	expectedFoo := []string{"1.0.0", "1.1.0"}
	if diff := cmp.Diff(expectedFoo, index["libfoo"]); diff != "" {
		t.Fatalf("unexpected libfoo versions (-want +got):\n%s", diff)
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
