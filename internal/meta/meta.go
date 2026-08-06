package meta

type PageMeta struct {
	Title       string
	Description string
	OGType      string
	OGImage     string
	Canonical   string
	NoIndex     bool
	NavKey      string
}

func New(title, description string) PageMeta {
	return PageMeta{
		Title:       title,
		Description: description,
		OGType:      "website",
	}
}

func (m PageMeta) WithOGImage(url string) PageMeta {
	m.OGImage = url
	return m
}

func (m PageMeta) WithCanonical(url string) PageMeta {
	m.Canonical = url
	return m
}

// WithNav marks which top-level nav item is current for this page.
func (m PageMeta) WithNav(key string) PageMeta {
	m.NavKey = key
	return m
}

func (m PageMeta) AsArticle() PageMeta {
	m.OGType = "article"
	return m
}
