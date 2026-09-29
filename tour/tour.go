package tour

import (
	"fmt"
	"io/fs"
	"sort"
	"strconv"
	"strings"
)

// Tour holds all chapters and lessons.
type Tour struct {
	Chapters []*Chapter
	lessons  []*Lesson // Flat list for navigation
}

// ParseTour parses a tour from the given filesystem.
// It expects markdown files in the root and lesson files in subdirectories.
func ParseTour(contentFS fs.FS) (*Tour, error) {
	tour := &Tour{}

	// Find all markdown files (chapters)
	entries, err := fs.ReadDir(contentFS, ".")
	if err != nil {
		return nil, fmt.Errorf("reading tour directory: %w", err)
	}

	var mdFiles []string
	for _, entry := range entries {
		name := entry.Name()
		// Skip README.md, DONE.md and other non-chapter files
		if !entry.IsDir() && strings.HasSuffix(name, ".md") && name != "README.md" && name != "DONE.md" {
			mdFiles = append(mdFiles, name)
		}
	}

	sort.Strings(mdFiles)

	for chapterIdx, mdFile := range mdFiles {
		chapter, err := parseChapter(contentFS, mdFile, chapterIdx)
		if err != nil {
			return nil, fmt.Errorf("parsing chapter %s: %w", mdFile, err)
		}
		tour.Chapters = append(tour.Chapters, chapter)
		tour.lessons = append(tour.lessons, chapter.Lessons...)
	}

	// Set navigation links
	for i, lesson := range tour.lessons {
		if i > 0 {
			prev := tour.lessons[i-1]
			lesson.HasPrev = true
			lesson.PrevID = prev.ID
			lesson.PrevSlug = prev.ChapterSlug
			lesson.PrevLessonIdx = prev.LessonIdx
		}
		if i < len(tour.lessons)-1 {
			next := tour.lessons[i+1]
			lesson.HasNext = true
			lesson.NextID = next.ID
			lesson.NextSlug = next.ChapterSlug
			lesson.NextLessonIdx = next.LessonIdx
		}
	}

	return tour, nil
}

// ValidateTour checks that all lessons in the tour have valid content.
func ValidateTour(t *Tour) error {
	for _, lesson := range t.lessons {
		if err := ValidateLesson(lesson); err != nil {
			return err
		}
	}
	return nil
}

// GetLesson returns a lesson by its ID.
func (t *Tour) GetLesson(id string) *Lesson {
	for _, lesson := range t.lessons {
		if lesson.ID == id {
			return lesson
		}
	}
	return nil
}

// GetLessonByName returns a lesson by chapter name and lesson index.
// The chapter name is matched against the suffix of the chapter name (e.g., "interpolation" matches "01-interpolation").
func (t *Tour) GetLessonByName(chapterName, lessonIdx string) *Lesson {
	for _, chapter := range t.Chapters {
		// Match friendly name to prefixed chapter name
		if chapter.Name == chapterName || chapter.Slug() == chapterName {
			idx, err := strconv.Atoi(lessonIdx)
			if err != nil || idx < 0 || idx >= len(chapter.Lessons) {
				return nil
			}
			return chapter.Lessons[idx]
		}
	}
	return nil
}

// FirstLesson returns the first lesson in the tour.
func (t *Tour) FirstLesson() *Lesson {
	if len(t.lessons) > 0 {
		return t.lessons[0]
	}
	return nil
}

// LessonCount returns the total number of lessons.
func (t *Tour) LessonCount() int {
	return len(t.lessons)
}
