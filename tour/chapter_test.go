package tour_test

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"

	"github.com/titpetric/vuego-cli/tour"
)

// TestChapter_Slug covers the name a chapter is reached by in a URL, which is
// its filename with the ordering prefix taken off.
func TestChapter_Slug(t *testing.T) {
	tests := map[string]string{
		"01-interpolation": "interpolation",
		"02-loops":         "loops",
		"intro":            "intro",
		"10-a-b-c":         "a-b-c",
		"":                 "",
	}

	for name, want := range tests {
		chapter := &tour.Chapter{Name: name}
		require.Equal(t, want, chapter.Slug(), "Slug() of %q", name)
	}
}

// TestChapter_SlugResolvesLessonDirectory covers what the slug is for: the
// chapter markdown is "01-basics.md" and the files it references live under
// "basics/", so a wrong slug loads nothing.
func TestChapter_SlugResolvesLessonDirectory(t *testing.T) {
	fs := fstest.MapFS{
		"01-basics.md": &fstest.MapFile{Data: []byte(`# Basics

## First Lesson

Content.

@file: index.vuego
`)},
		"basics/index.vuego": &fstest.MapFile{Data: []byte(`<div>Hello</div>`)},
	}

	parsed, err := tour.ParseTour(fs)
	require.NoError(t, err)
	require.Len(t, parsed.Chapters, 1)

	chapter := parsed.Chapters[0]
	require.Equal(t, "01-basics", chapter.Name)
	require.Equal(t, "basics", chapter.Slug())

	lesson := parsed.GetLesson("0/0")
	require.NotNil(t, lesson)
	require.Equal(t, "basics", lesson.ChapterSlug)
	require.Equal(t, "<div>Hello</div>", lesson.Files["index.vuego"])
}
