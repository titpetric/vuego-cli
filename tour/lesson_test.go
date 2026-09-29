package tour_test

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"

	"github.com/titpetric/vuego-cli/tour"
)

func TestLesson_PrimaryTemplate(t *testing.T) {
	t.Run("returns index.vuego when present", func(t *testing.T) {
		fs := fstest.MapFS{
			"01-basics.md": &fstest.MapFile{
				Data: []byte(`# Basics

## First Lesson

Content.

@file: index.vuego
@file: other.vuego
`),
			},
			"basics/index.vuego": &fstest.MapFile{
				Data: []byte(`<div>Hello</div>`),
			},
			"basics/other.vuego": &fstest.MapFile{
				Data: []byte(`<div>Other</div>`),
			},
		}

		parsed, err := tour.ParseTour(fs)
		require.NoError(t, err)
		require.Len(t, parsed.Chapters, 1)

		lesson := parsed.GetLesson("0/0")
		require.NotNil(t, lesson)
		require.Equal(t, "index.vuego", lesson.PrimaryTemplate())
	})

	t.Run("returns first vuego file when no index.vuego", func(t *testing.T) {
		fs := fstest.MapFS{
			"01-basics.md": &fstest.MapFile{
				Data: []byte(`# Basics

## First Lesson

Content.

@file: template.vuego
`),
			},
			"basics/template.vuego": &fstest.MapFile{
				Data: []byte(`<div>Hello</div>`),
			},
		}

		parsed, err := tour.ParseTour(fs)
		require.NoError(t, err)

		lesson := parsed.GetLesson("0/0")
		require.NotNil(t, lesson)
		require.Equal(t, "template.vuego", lesson.PrimaryTemplate())
	})

	t.Run("returns empty string when no vuego files", func(t *testing.T) {
		fs := fstest.MapFS{
			"01-basics.md": &fstest.MapFile{
				Data: []byte(`# Basics

## First Lesson

Content.

@file: data.json
`),
			},
			"basics/data.json": &fstest.MapFile{
				Data: []byte(`{"name": "test"}`),
			},
		}

		parsed, err := tour.ParseTour(fs)
		require.NoError(t, err)

		lesson := parsed.GetLesson("0/0")
		require.NotNil(t, lesson)
		require.Equal(t, "", lesson.PrimaryTemplate())
	})
}

func TestLesson_DataFile(t *testing.T) {
	t.Run("returns index.yml when index.vuego and index.yml exist", func(t *testing.T) {
		fs := fstest.MapFS{
			"01-basics.md": &fstest.MapFile{
				Data: []byte(`# Basics

## First Lesson

Content.

@file: index.vuego
@file: index.yml
`),
			},
			"basics/index.vuego": &fstest.MapFile{
				Data: []byte(`<div>Hello</div>`),
			},
			"basics/index.yml": &fstest.MapFile{
				Data: []byte(`name: test`),
			},
		}

		parsed, err := tour.ParseTour(fs)
		require.NoError(t, err)

		lesson := parsed.GetLesson("0/0")
		require.NotNil(t, lesson)
		require.Equal(t, "index.yml", lesson.DataFile())
	})

	t.Run("returns index.json when index.vuego and index.json exist", func(t *testing.T) {
		fs := fstest.MapFS{
			"01-basics.md": &fstest.MapFile{
				Data: []byte(`# Basics

## First Lesson

Content.

@file: index.vuego
@file: index.json
`),
			},
			"basics/index.vuego": &fstest.MapFile{
				Data: []byte(`<div>Hello</div>`),
			},
			"basics/index.json": &fstest.MapFile{
				Data: []byte(`{"name": "test"}`),
			},
		}

		parsed, err := tour.ParseTour(fs)
		require.NoError(t, err)

		lesson := parsed.GetLesson("0/0")
		require.NotNil(t, lesson)
		require.Equal(t, "index.json", lesson.DataFile())
	})

	t.Run("returns empty string when no matching data file", func(t *testing.T) {
		fs := fstest.MapFS{
			"01-basics.md": &fstest.MapFile{
				Data: []byte(`# Basics

## First Lesson

Content.

@file: index.vuego
@file: other.json
`),
			},
			"basics/index.vuego": &fstest.MapFile{
				Data: []byte(`<div>Hello</div>`),
			},
			"basics/other.json": &fstest.MapFile{
				Data: []byte(`{"name": "test"}`),
			},
		}

		parsed, err := tour.ParseTour(fs)
		require.NoError(t, err)

		lesson := parsed.GetLesson("0/0")
		require.NotNil(t, lesson)
		require.Equal(t, "", lesson.DataFile())
	})

	t.Run("returns empty string when no primary template", func(t *testing.T) {
		fs := fstest.MapFS{
			"01-basics.md": &fstest.MapFile{
				Data: []byte(`# Basics

## First Lesson

Content.

@file: index.json
`),
			},
			"basics/index.json": &fstest.MapFile{
				Data: []byte(`{"name": "test"}`),
			},
		}

		parsed, err := tour.ParseTour(fs)
		require.NoError(t, err)

		lesson := parsed.GetLesson("0/0")
		require.NotNil(t, lesson)
		require.Equal(t, "", lesson.DataFile())
	})
}
func TestLesson_PrimaryTemplate_Direct(t *testing.T) {
	t.Run("index.vuego returns index.vuego", func(t *testing.T) {
		lesson := &tour.Lesson{
			Files: map[string]string{
				"index.vuego": "<div>Hello</div>",
				"other.vuego": "<div>Other</div>",
			},
		}
		require.Equal(t, "index.vuego", lesson.PrimaryTemplate())
	})

	t.Run("only other.vuego returns other.vuego", func(t *testing.T) {
		lesson := &tour.Lesson{
			Files: map[string]string{
				"other.vuego": "<div>Other</div>",
			},
		}
		require.Equal(t, "other.vuego", lesson.PrimaryTemplate())
	})

	t.Run("no vuego files returns empty string", func(t *testing.T) {
		lesson := &tour.Lesson{
			Files: map[string]string{
				"data.json": `{"key": "value"}`,
			},
		}
		require.Equal(t, "", lesson.PrimaryTemplate())
	})
}

func TestLesson_DataFile_Direct(t *testing.T) {
	t.Run("index.vuego + index.yml returns index.yml", func(t *testing.T) {
		lesson := &tour.Lesson{
			Files: map[string]string{
				"index.vuego": "<div>Hello</div>",
				"index.yml":   "key: value",
			},
		}
		require.Equal(t, "index.yml", lesson.DataFile())
	})

	t.Run("index.vuego + index.json returns index.json", func(t *testing.T) {
		lesson := &tour.Lesson{
			Files: map[string]string{
				"index.vuego": "<div>Hello</div>",
				"index.json":  `{"key": "value"}`,
			},
		}
		require.Equal(t, "index.json", lesson.DataFile())
	})

	t.Run("index.vuego + no data file returns empty string", func(t *testing.T) {
		lesson := &tour.Lesson{
			Files: map[string]string{
				"index.vuego": "<div>Hello</div>",
			},
		}
		require.Equal(t, "", lesson.DataFile())
	})

	t.Run("no primary template returns empty string", func(t *testing.T) {
		lesson := &tour.Lesson{
			Files: map[string]string{
				"index.json": `{"key": "value"}`,
			},
		}
		require.Equal(t, "", lesson.DataFile())
	})
}

// TestValidateLesson covers what makes a lesson worth showing: something the
// reader can run. A lesson gets its runnable file either from an @file
// reference or from a fenced block, and both count.
func TestValidateLesson(t *testing.T) {
	tests := []struct {
		name    string
		lesson  *tour.Lesson
		wantErr bool
	}{
		{"vuego reference", &tour.Lesson{FileRefs: []string{"index.vuego"}}, false},
		{"php reference", &tour.Lesson{FileRefs: []string{"index.php"}}, false},
		{"sql reference", &tour.Lesson{FileRefs: []string{"query.sql"}}, false},
		{"inline fence", &tour.Lesson{Files: map[string]string{"index.php": "<?php"}}, false},
		{"data only", &tour.Lesson{
			Title:    "Prose",
			Chapter:  "01-intro",
			FileRefs: []string{"notes.json"},
			Files:    map[string]string{"notes.json": "{}"},
		}, true},
		{"nothing at all", &tour.Lesson{Title: "Empty", Chapter: "01-intro"}, true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := tour.ValidateLesson(test.lesson)
			if !test.wantErr {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			require.Contains(t, err.Error(), test.lesson.Title, "the error names the lesson")
			require.Contains(t, err.Error(), test.lesson.Chapter, "and the chapter it is in")
		})
	}
}
