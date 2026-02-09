package adapters

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"avular-packages/internal/types"
)

const testPackageXMLWithROSTags = `<?xml version="1.0"?>
<package format="3">
  <name>my_node</name>
  <version>1.0.0</version>
  <description>Test package</description>

  <depend>rclcpp</depend>
  <depend>std_msgs</depend>

  <exec_depend>fmt</exec_depend>
  <exec_depend>opencv</exec_depend>

  <build_depend>ament_cmake</build_depend>
  <build_depend>rosidl_default_generators</build_depend>

  <build_export_depend>rosidl_default_runtime</build_export_depend>

  <test_depend>ament_lint_auto</test_depend>

  <export>
    <debian_depend>libfmt-dev</debian_depend>
    <pip_depend version="3.1.2">flask</pip_depend>
  </export>
</package>
`

func TestParseROSTags(t *testing.T) {
	dir := t.TempDir()
	xmlPath := filepath.Join(dir, "package.xml")
	require.NoError(t, os.WriteFile(xmlPath, []byte(testPackageXMLWithROSTags), 0644))

	adapter := NewPackageXMLAdapter()
	tags, err := adapter.ParseROSTags([]string{xmlPath})
	require.NoError(t, err)

	// Build expected set
	expected := map[string]types.ROSDepScope{
		"rclcpp":                    types.ROSDepScopeAll,
		"std_msgs":                  types.ROSDepScopeAll,
		"fmt":                       types.ROSDepScopeExec,
		"opencv":                    types.ROSDepScopeExec,
		"ament_cmake":               types.ROSDepScopeBuild,
		"rosidl_default_generators": types.ROSDepScopeBuild,
		"rosidl_default_runtime":    types.ROSDepScopeBuildExec,
		"ament_lint_auto":           types.ROSDepScopeTest,
	}

	assert.Len(t, tags, len(expected))
	for _, tag := range tags {
		expectedScope, ok := expected[tag.Key]
		if !ok {
			t.Errorf("unexpected tag key: %s", tag.Key)
			continue
		}
		assert.Equal(t, expectedScope, tag.Scope, "wrong scope for %s", tag.Key)
	}
}

func TestParseROSTagsCoexistsWithExportTags(t *testing.T) {
	dir := t.TempDir()
	xmlPath := filepath.Join(dir, "package.xml")
	require.NoError(t, os.WriteFile(xmlPath, []byte(testPackageXMLWithROSTags), 0644))

	adapter := NewPackageXMLAdapter()

	// ROS tags should work
	tags, err := adapter.ParseROSTags([]string{xmlPath})
	require.NoError(t, err)
	assert.NotEmpty(t, tags)

	// Export tags should still work
	debs, pips, err := adapter.ParseDependencies([]string{xmlPath}, []string{"debian_depend", "pip_depend"})
	require.NoError(t, err)
	assert.Equal(t, []string{"libfmt-dev"}, debs)
	assert.Equal(t, []string{"flask==3.1.2"}, pips)
}

func TestParseROSTagsEmptyXML(t *testing.T) {
	dir := t.TempDir()
	xmlPath := filepath.Join(dir, "package.xml")
	require.NoError(t, os.WriteFile(xmlPath, []byte(`<?xml version="1.0"?>
<package format="3">
  <name>empty_pkg</name>
  <version>0.0.1</version>
</package>`), 0644))

	adapter := NewPackageXMLAdapter()
	tags, err := adapter.ParseROSTags([]string{xmlPath})
	require.NoError(t, err)
	assert.Empty(t, tags)
}
