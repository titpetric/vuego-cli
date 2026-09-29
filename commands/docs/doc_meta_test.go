package docs_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v3"

	"github.com/titpetric/vuego-cli/commands/docs"
)

// TestDocMeta covers the front matter keys a document is allowed to set. The
// names are what a .md file writes, so they are part of the content format
// rather than an implementation detail: renaming a field renames the key every
// document already uses.
func TestDocMeta(t *testing.T) {
	var meta docs.DocMeta

	err := yaml.Unmarshal([]byte("title: Hello\nsubtitle: A subtitle\nlayout: page\n"), &meta)

	require.NoError(t, err)
	require.Equal(t, "Hello", meta.Title)
	require.Equal(t, "A subtitle", meta.Subtitle)
	require.Equal(t, "page", meta.Layout)
}

// TestDocMetaPartialFrontMatter covers a document setting only some keys,
// which is the common case: the rest stay empty rather than failing the parse.
func TestDocMetaPartialFrontMatter(t *testing.T) {
	var meta docs.DocMeta

	err := yaml.Unmarshal([]byte("title: Only a title\n"), &meta)

	require.NoError(t, err)
	require.Equal(t, "Only a title", meta.Title)
	require.Equal(t, "", meta.Subtitle)
	require.Equal(t, "", meta.Layout)
}
