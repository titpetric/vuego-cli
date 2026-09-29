package tour

import (
	"fmt"
	"strings"
)

// Lesson represents a single lesson within a chapter.
type Lesson struct {
	ID             string              // Unique identifier (chapter/lesson index)
	Title          string              // Lesson title from ## heading
	Content        string              // Markdown content for the lesson
	Files          map[string]string   // Template and data files for this lesson
	FileRefs       []string            // File paths from @file: references
	FileOptions    map[string][]string // Optional hints from @file references, keyed by filename
	Chapter        string              // Parent chapter name
	ChapterTitle   string              // Parent chapter title (friendly name)
	ChapterSlug    string              // URL-friendly chapter name (e.g., "interpolation")
	ChapterIdx     int                 // Chapter index (0-based)
	LessonIdx      int                 // Lesson index within chapter (0-based)
	HasPrev        bool                // Whether there's a previous lesson
	HasNext        bool                // Whether there's a next lesson
	PrevID         string              // Previous lesson ID
	NextID         string              // Next lesson ID
	PrevSlug       string              // Previous lesson chapter slug
	PrevLessonIdx  int                 // Previous lesson index
	NextSlug       string              // Next lesson chapter slug
	NextLessonIdx  int                 // Next lesson index
	TotalInChapter int                 // Total lessons in this chapter
}

// PrimaryTemplate returns the main .vuego template filename for the lesson.
// It looks for index.vuego first, then any .vuego file.
func (l *Lesson) PrimaryTemplate() string {
	if _, ok := l.Files["index.vuego"]; ok {
		return "index.vuego"
	}
	for name := range l.Files {
		if strings.HasSuffix(name, ".vuego") {
			return name
		}
	}
	return ""
}

// DataFile returns the data filename (.yaml or .json) that matches the primary template.
func (l *Lesson) DataFile() string {
	primary := l.PrimaryTemplate()
	if primary == "" {
		return ""
	}
	base := strings.TrimSuffix(primary, ".vuego")
	for _, ext := range []string{".yaml", ".yml", ".json"} {
		if _, ok := l.Files[base+ext]; ok {
			return base + ext
		}
	}
	return ""
}

// ValidateLesson checks that a lesson has at least one runnable file.
func ValidateLesson(lesson *Lesson) error {
	for _, ref := range lesson.FileRefs {
		if isRunnableFile(ref) {
			return nil
		}
	}
	for name := range lesson.Files {
		if isRunnableFile(name) {
			return nil
		}
	}
	return fmt.Errorf("lesson %q in chapter %q has no runnable file", lesson.Title, lesson.Chapter)
}

func isRunnableFile(name string) bool {
	return strings.HasSuffix(name, ".vuego") || strings.HasSuffix(name, ".php") || strings.HasSuffix(name, ".sql")
}
