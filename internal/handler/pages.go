package handler

import (
	"net/http"

	"detent.build/internal/content"
	"detent.build/templates/pages"
	"detent.build/templates/partials"

	"github.com/labstack/echo/v4"
)

func (h *Handler) Home(c echo.Context) error {
	return render(c, http.StatusOK, pages.Home())
}

func (h *Handler) HowItWorks(c echo.Context) error {
	return render(c, http.StatusOK, pages.HowItWorks())
}

func (h *Handler) WhyDetent(c echo.Context) error {
	return render(c, http.StatusOK, pages.WhyDetent())
}

func (h *Handler) Dashboard(c echo.Context) error {
	return render(c, http.StatusOK, pages.Dashboard())
}

func (h *Handler) Install(c echo.Context) error {
	return render(c, http.StatusOK, pages.Install(content.InstallTargets[0]))
}

// InstallPlatform serves one platform's install commands. HTMX swaps the tab
// strip along with the panel so the selected state stays truthful; a direct
// visit renders the whole page with that tab selected, so every tab is
// linkable and works with JavaScript off.
func (h *Handler) InstallPlatform(c echo.Context) error {
	target, ok := findTarget(c.Param("platform"))
	if !ok {
		return echo.NewHTTPError(http.StatusNotFound, "unknown platform")
	}

	if c.Request().Header.Get("HX-Request") == "true" {
		return render(c, http.StatusOK, partials.InstallTabs(target))
	}
	return render(c, http.StatusOK, pages.Install(target))
}

func (h *Handler) OpenSource(c echo.Context) error {
	return render(c, http.StatusOK, pages.OpenSource())
}

func findTarget(key string) (content.InstallTarget, bool) {
	for _, t := range content.InstallTargets {
		if t.Key == key {
			return t, true
		}
	}
	return content.InstallTarget{}, false
}
