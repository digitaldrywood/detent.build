package meta

type PageMeta struct {
	Title       string
	Description string
	OGType      string
	OGImage     string
	// Path is the page's site-root path, e.g. "/how-it-works". The canonical
	// and og:url are resolved from it against the configured site URL, so the
	// production host lives in one place (SITE_URL) rather than being repeated
	// in every page's metadata.
	Path    string
	NoIndex bool
	NavKey  string
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

// WithPath sets the site-root path the canonical and og:url resolve from.
func (m PageMeta) WithPath(path string) PageMeta {
	m.Path = path
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
