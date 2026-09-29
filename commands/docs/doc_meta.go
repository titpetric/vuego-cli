package docs

// DocMeta represents frontmatter metadata for a doc.
type DocMeta struct {
	Title    string `yaml:"title"`
	Subtitle string `yaml:"subtitle"`
	Layout   string `yaml:"layout"`
}
