package tour

import (
	"fmt"
	"io/fs"
	"path"
	"strings"
)

// Chapter represents a chapter containing multiple lessons.
type Chapter struct {
	Name    string    // Chapter name (from filename)
	Title   string    // Chapter title (from # heading or filename)
	Lessons []*Lesson // Lessons in this chapter
	Index   int       // Chapter index (0-based)
}

// Slug returns the directory name for a chapter (strips numeric prefix).
// e.g., "01-interpolation" -> "interpolation".
func (c *Chapter) Slug() string {
	parts := strings.SplitN(c.Name, "-", 2)
	if len(parts) == 2 {
		return parts[1]
	}
	return c.Name
}

func parseChapter(contentFS fs.FS, mdFile string, chapterIdx int) (*Chapter, error) {
	content, err := fs.ReadFile(contentFS, mdFile)
	if err != nil {
		return nil, err
	}

	chapterName := strings.TrimSuffix(mdFile, ".md")
	chapter := &Chapter{
		Name:  chapterName,
		Title: chapterName,
		Index: chapterIdx,
	}

	// Parse markdown into lessons (split by --- delimiter)
	lessons, chapterTitle := chapter.parseMarkdownLessons(string(content))
	if chapterTitle != "" {
		chapter.Title = chapterTitle
	}

	// Load files for each lesson from @file references
	for _, lesson := range lessons {
		for name, content := range chapter.loadLessonFilesFromRefs(contentFS, lesson.FileRefs) {
			lesson.Files[name] = content
		}
		lesson.TotalInChapter = len(lessons)
		lesson.ChapterTitle = chapter.Title
		lesson.ChapterSlug = chapter.Slug()
	}

	chapter.Lessons = lessons
	return chapter, nil
}

func (c *Chapter) parseMarkdownLessons(content string) ([]*Lesson, string) {
	var lessons []*Lesson
	var chapterTitle string
	var chapterName = c.Name
	var chapterIdx = c.Index

	// Split by lesson delimiter: \n\n---\n\n
	sections := strings.Split(content, "\n\n---\n\n")

	for lessonIdx, section := range sections {
		section = strings.TrimSpace(section)
		if section == "" {
			continue
		}

		// First section may contain chapter title (# heading)
		lines := strings.SplitN(section, "\n", 2)
		title := ""
		lessonContent := section

		if len(lines) > 0 && strings.HasPrefix(lines[0], "# ") {
			// This is the chapter title, extract it
			if chapterTitle == "" {
				chapterTitle = strings.TrimPrefix(lines[0], "# ")
			}
			if len(lines) > 1 {
				lessonContent = strings.TrimSpace(lines[1])
			} else {
				continue // Only chapter title, no lesson content
			}
		}

		// Extract lesson title from ## heading if present
		contentLines := strings.SplitN(lessonContent, "\n", 2)
		if len(contentLines) > 0 && strings.HasPrefix(contentLines[0], "## ") {
			title = strings.TrimPrefix(contentLines[0], "## ")
			if len(contentLines) > 1 {
				lessonContent = strings.TrimSpace(contentLines[1])
			} else {
				lessonContent = ""
			}
		}

		if title == "" {
			title = fmt.Sprintf("Lesson %d", lessonIdx+1)
		}

		// Extract @file: references and runnable fenced code blocks from content
		fileRefs, fileOptions, cleanContent := extractFileRefs(lessonContent)
		inlineFiles, cleanContent := extractRunnableCodeBlocks(cleanContent)

		lesson := &Lesson{
			ID:          fmt.Sprintf("%d/%d", chapterIdx, len(lessons)),
			Title:       title,
			Content:     cleanContent,
			FileRefs:    fileRefs,
			FileOptions: fileOptions,
			Chapter:     chapterName,
			ChapterIdx:  chapterIdx,
			LessonIdx:   len(lessons),
			Files:       inlineFiles,
		}
		lessons = append(lessons, lesson)
	}

	return lessons, chapterTitle
}

// extractFileRefs extracts @file: references from lesson content.
// Returns file paths, optional hints keyed by filename, and content with @file lines removed.
func extractFileRefs(content string) ([]string, map[string][]string, string) {
	var refs []string
	options := make(map[string][]string)
	var cleanLines []string

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "@file:") {
			fields := strings.Fields(strings.TrimSpace(strings.TrimPrefix(trimmed, "@file:")))
			filePath := ""
			if len(fields) > 0 {
				filePath = fields[0]
			}
			if filePath != "" {
				refs = append(refs, filePath)
				if len(fields) > 1 {
					options[path.Base(filePath)] = fields[1:]
				}
			}
		} else {
			cleanLines = append(cleanLines, line)
		}
	}

	return refs, options, strings.TrimSpace(strings.Join(cleanLines, "\n"))
}

func extractRunnableCodeBlocks(content string) (map[string]string, string) {
	files := make(map[string]string)
	var cleanLines []string
	lines := strings.Split(content, "\n")
	counters := map[string]int{}

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "```") && !strings.HasPrefix(trimmed, "~~~") {
			cleanLines = append(cleanLines, line)
			continue
		}

		fence := trimmed[:3]
		info := strings.TrimSpace(trimmed[3:])
		ext := runnableExt(info)
		if ext == "" {
			cleanLines = append(cleanLines, line)
			continue
		}

		var block []string
		for i++; i < len(lines); i++ {
			if strings.HasPrefix(strings.TrimSpace(lines[i]), fence) {
				break
			}
			block = append(block, lines[i])
		}

		counters[ext]++
		name := "index" + ext
		if counters[ext] > 1 || files[name] != "" {
			name = fmt.Sprintf("snippet-%d%s", counters[ext], ext)
		}
		files[name] = strings.Join(block, "\n")
		cleanLines = append(cleanLines, fmt.Sprintf("```%s", strings.TrimPrefix(ext, ".")))
		cleanLines = append(cleanLines, block...)
		cleanLines = append(cleanLines, "```")
		cleanLines = append(cleanLines, "")
		cleanLines = append(cleanLines, fmt.Sprintf("_Runnable as `%s`._", name))
	}

	return files, strings.TrimSpace(strings.Join(cleanLines, "\n"))
}

func runnableExt(info string) string {
	fields := strings.Fields(info)
	if len(fields) == 0 {
		return ""
	}
	switch strings.ToLower(fields[0]) {
	case "php", "application/x-httpd-php":
		return ".php"
	case "vuego", "html+vuego":
		return ".vuego"
	case "json":
		return ".json"
	case "yaml", "yml":
		return ".yaml"
	case "sql", "sqlite", "sqlite3":
		return ".sql"
	default:
		return ""
	}
}

// loadLessonFilesFromRefs loads files from @file: references.
// It also implicitly loads data files (.yml, .yaml, .json) based on the primary template name.
func (c *Chapter) loadLessonFilesFromRefs(contentFS fs.FS, refs []string) map[string]string {
	files := make(map[string]string)
	dir := c.Slug()

	for _, ref := range refs {
		// Build full path relative to chapter directory (without numeric prefix)
		filePath := path.Join(dir, ref)
		content, err := fs.ReadFile(contentFS, filePath)
		if err != nil {
			continue
		}
		// Use just the filename as the key
		fileName := path.Base(ref)
		files[fileName] = string(content)

		// Implicitly load data file for .vuego templates
		if strings.HasSuffix(fileName, ".vuego") {
			basePath := strings.TrimSuffix(filePath, ".vuego")
			baseName := strings.TrimSuffix(fileName, ".vuego")
			for _, ext := range []string{".yaml", ".yml", ".json"} {
				dataPath := basePath + ext
				dataContent, err := fs.ReadFile(contentFS, dataPath)
				if err == nil {
					files[baseName+ext] = string(dataContent)
					break
				}
			}
		}

		// Implicitly load companion migration for .sql examples.
		if strings.HasSuffix(fileName, ".sql") {
			basePath := strings.TrimSuffix(filePath, ".sql")
			baseName := strings.TrimSuffix(fileName, ".sql")
			for _, ext := range []string{".up.sql", ".sqlite3"} {
				dataPath := basePath + ext
				dataContent, err := fs.ReadFile(contentFS, dataPath)
				if err == nil {
					files[baseName+ext] = string(dataContent)
					break
				}
			}
		}
	}

	return files
}
