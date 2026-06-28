package handlers

import (
	"embed"
	"encoding/json"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"github.com/MatejGadjo/NewsAggregator/internal/logger"
	"github.com/gin-gonic/gin"
)

// canonicalBaseURL is the apex origin used for canonical/og URLs. The www host
// 301s here (see PRODUCTION.md). Keep in sync if the domain changes.
const canonicalBaseURL = "https://jadro.mk"

// defaultOGImage is the fallback share image when a cluster has no article image.
const defaultOGImage = canonicalBaseURL + "/og-image.png"

const metaDescriptionMax = 160

//go:embed templates/*.gohtml
var prerenderTemplates embed.FS

// staticPaths are real non-topic SPA routes that must render a valid 200 page
// rather than a 404 when they hit the topic branch. Mirrors the static routes
// in frontend src/routes.tsx.
var staticPaths = map[string]struct{}{
	"about":   {},
	"privacy": {},
	"terms":   {},
}

// topicSlugToValue maps the latin URL slug to the Cyrillic topic value the feed
// service expects. Mirrors frontend src/lib/topics.ts (TOPIC_ITEMS).
var topicSlugToValue = map[string]string{
	"makedonija":  "Македонија",
	"hronika":     "Хроника",
	"svet":        "Свет",
	"ekonomija":   "Економија",
	"sport":       "Спорт",
	"skopje":      "Скопје",
	"region":      "Регион",
	"tehnologija": "Технологија",
	"kultura":     "Култура",
	"scena":       "Сцена",
}

// PrerenderHandler serves crawler-only, server-rendered HTML for page routes.
// Real users are routed to the SPA by nginx; only bot user-agents reach here.
// It reuses the existing (cached) feed/cluster JSON — no extra DB queries — and
// renders with html/template so untrusted RSS content is auto-escaped.
type PrerenderHandler struct {
	feed    FeedServicer
	cluster ClusterServicer
	tmpl    *template.Template
	log     *logger.Logger
}

func NewPrerenderHandler(feed FeedServicer, cluster ClusterServicer, log *logger.Logger) *PrerenderHandler {
	tmpl := template.Must(template.ParseFS(prerenderTemplates, "templates/*.gohtml"))
	return &PrerenderHandler{feed: feed, cluster: cluster, tmpl: tmpl, log: log}
}

// --- minimal unmarshal shapes (only the stable fields we render) ---

type prCluster struct {
	ID           int64  `json:"id"`
	Headline     string `json:"headline"`
	DisplayTitle string `json:"display_title"`
	Summary      string `json:"summary"`
	Topic        string `json:"topic"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

func (c prCluster) title() string {
	if strings.TrimSpace(c.DisplayTitle) != "" {
		return c.DisplayTitle
	}
	return c.Headline
}

type prSource struct {
	Name    string `json:"name"`
	BaseURL string `json:"base_url"`
}

type prArticle struct {
	URL      string   `json:"url"`
	ImageURL *string  `json:"image_url"`
	Title    string   `json:"title"`
	Snippet  string   `json:"snippet"`
	Source   prSource `json:"source"`
}

type prClusterResponse struct {
	Cluster  prCluster   `json:"cluster"`
	Articles []prArticle `json:"articles"`
}

type prClusterDetail struct {
	Cluster  prCluster   `json:"cluster"`
	Articles []prArticle `json:"articles"`
}

type prFeedResponse struct {
	Clusters       []prCluster       `json:"clusters"`
	ClusterDetails []prClusterDetail `json:"cluster_details"`
}

// --- view models passed to templates ---

type sourceLink struct {
	Name string
	URL  string
}

type listItem struct {
	Headline string
	Summary  string
	URL      string
}

type clusterView struct {
	Title       string
	Headline    string
	Description string
	Canonical   string
	OGImage     string
	Published   string
	Modified    string
	Sources     []sourceLink
	JSONLD      template.JS
}

type listView struct {
	Title       string
	Heading     string
	Description string
	Canonical   string
	OGImage     string
	Items       []listItem
}

type notFoundView struct {
	Title       string
	Heading     string
	Description string
	Canonical   string
}

// Render is the single catch-all entry point. nginx forwards the original page
// path appended to /_prerender, so c.Param("path") is e.g. "/cluster/123",
// "/sport", or "/". Unknown paths fall back to a generic site page (always
// valid HTML, never a hard error for a crawler).
func (h *PrerenderHandler) Render(c *gin.Context) {
	path := c.Param("path")
	switch {
	case path == "" || path == "/":
		h.renderHome(c)
	case strings.HasPrefix(path, "/cluster/"):
		h.renderCluster(c, strings.TrimPrefix(path, "/cluster/"))
	default:
		h.renderTopic(c, strings.Trim(path, "/"))
	}
}

func (h *PrerenderHandler) renderCluster(c *gin.Context, idStr string) {
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		h.renderNotFound(c)
		return
	}

	raw, status, err := h.cluster.GetCluster(c.Request.Context(), id)
	if err != nil || status != http.StatusOK {
		// A missing cluster must return 404 so Google doesn't index an empty
		// page; a backend error also has no content to render.
		h.renderNotFound(c)
		return
	}

	var resp prClusterResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		h.log.Warnw("prerender: cluster unmarshal failed", "cluster_id", id, "error", err)
		h.renderNotFound(c)
		return
	}

	cl := resp.Cluster
	canonical := canonicalBaseURL + "/cluster/" + idStr
	ogImage := firstImage(resp.Articles)
	if ogImage == "" {
		ogImage = defaultOGImage
	}

	sources := make([]sourceLink, 0, len(resp.Articles))
	citations := make([]string, 0, len(resp.Articles))
	for _, a := range resp.Articles {
		name := a.Source.Name
		if name == "" {
			name = a.Title
		}
		sources = append(sources, sourceLink{Name: name, URL: a.URL})
		citations = append(citations, a.URL)
	}

	view := clusterView{
		Title:       cl.title() + " — Јадро",
		Headline:    cl.title(),
		Description: truncate(cl.Summary, metaDescriptionMax),
		Canonical:   canonical,
		OGImage:     ogImage,
		Published:   cl.CreatedAt,
		Modified:    cl.UpdatedAt,
		Sources:     sources,
		JSONLD:      newsArticleJSONLD(cl, canonical, ogImage, citations),
	}
	h.execute(c, http.StatusOK, "cluster.gohtml", view)
}

func (h *PrerenderHandler) renderTopic(c *gin.Context, slug string) {
	value, ok := topicSlugToValue[strings.ToLower(slug)]
	if !ok {
		if _, isStatic := staticPaths[strings.ToLower(slug)]; isStatic {
			// Real non-topic SPA page (e.g. /about) — serve a valid 200 page.
			h.renderFallback(c)
			return
		}
		// Unknown path — return a real 404 so crawlers don't index an empty SPA
		// route (soft-404).
		h.renderNotFound(c)
		return
	}

	raw, err := h.feed.GetFeed(c.Request.Context(), &value, 0)
	if err != nil {
		h.renderFallback(c)
		return
	}
	view := h.feedToList(raw,
		value+" — Вести — Јадро",
		value,
		"Најнови вести од категоријата "+value+" на Јадро — собрани од сите македонски извори.",
		canonicalBaseURL+"/"+slug,
	)
	h.execute(c, http.StatusOK, "list.gohtml", view)
}

func (h *PrerenderHandler) renderHome(c *gin.Context) {
	raw, err := h.feed.GetFeed(c.Request.Context(), nil, 0)
	if err != nil {
		h.renderFallback(c)
		return
	}
	view := h.feedToList(raw,
		"Јадро — Македонски Новинарски Агрегатор",
		"Денес во Македонија",
		"Најважните македонски вести од сите извори на едно место, без хаос.",
		canonicalBaseURL+"/",
	)
	h.execute(c, http.StatusOK, "list.gohtml", view)
}

func (h *PrerenderHandler) feedToList(raw []byte, title, heading, description, canonical string) listView {
	var resp prFeedResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		h.log.Warnw("prerender: feed unmarshal failed", "error", err)
	}

	items := make([]listItem, 0, len(resp.Clusters))
	for _, cl := range resp.Clusters {
		items = append(items, listItem{
			Headline: cl.title(),
			Summary:  truncate(cl.Summary, metaDescriptionMax),
			URL:      canonicalBaseURL + "/cluster/" + strconv.FormatInt(cl.ID, 10),
		})
	}

	// Homepage and topic-list pages represent the brand/category, not a single
	// story, so they always share the branded card. A per-article image here
	// meant sharing jadro.mk surfaced whatever cluster happened to be on top.
	// Only cluster detail pages (renderCluster) use an article image.
	return listView{
		Title:       title,
		Heading:     heading,
		Description: description,
		Canonical:   canonical,
		OGImage:     defaultOGImage,
		Items:       items,
	}
}

// renderFallback serves a minimal valid 200 page for known-but-errored routes
// (e.g. /about, or a topic whose feed query failed) so a crawler always gets
// well-formed HTML pointing back to the homepage.
func (h *PrerenderHandler) renderFallback(c *gin.Context) {
	view := listView{
		Title:       "Јадро — Македонски Новинарски Агрегатор",
		Heading:     "Јадро",
		Description: "Македонски агрегатор на вести. Најважните вести од сите извори на едно место.",
		Canonical:   canonicalBaseURL + "/",
		OGImage:     defaultOGImage,
		Items:       nil,
	}
	h.execute(c, http.StatusOK, "list.gohtml", view)
}

// renderNotFound serves the branded 404 page with a real HTTP 404 status, so
// crawlers don't index nonexistent paths (missing clusters, unknown slugs). The
// template is noindex as a second signal.
func (h *PrerenderHandler) renderNotFound(c *gin.Context) {
	view := notFoundView{
		Title:       "Страницата не постои — Јадро",
		Heading:     "404",
		Description: "Страницата што ја баравте не постои или е преместена.",
		Canonical:   canonicalBaseURL + "/",
	}
	h.execute(c, http.StatusNotFound, "notfound.gohtml", view)
}

func (h *PrerenderHandler) execute(c *gin.Context, status int, name string, data any) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Status(status)
	if err := h.tmpl.ExecuteTemplate(c.Writer, name, data); err != nil {
		h.log.Errorw("prerender: template execute failed", "template", name, "error", err)
		// Best effort — only takes effect if nothing has been written yet.
		c.Status(http.StatusInternalServerError)
	}
}

// newsArticleJSONLD builds a schema.org NewsArticle. encoding/json HTML-escapes
// <, >, & by default, so the result is safe to embed in a <script> tag.
func newsArticleJSONLD(cl prCluster, canonical, ogImage string, citations []string) template.JS {
	doc := map[string]any{
		"@context":         "https://schema.org",
		"@type":            "NewsArticle",
		"headline":         cl.title(),
		"description":      truncate(cl.Summary, metaDescriptionMax),
		"inLanguage":       "mk",
		"mainEntityOfPage": canonical,
		"image":            []string{ogImage},
		"publisher": map[string]any{
			"@type": "NewsMediaOrganization",
			"name":  "Јадро",
			"url":   canonicalBaseURL,
		},
	}
	if cl.CreatedAt != "" {
		doc["datePublished"] = cl.CreatedAt
	}
	if cl.UpdatedAt != "" {
		doc["dateModified"] = cl.UpdatedAt
	}
	if len(citations) > 0 {
		doc["citation"] = citations
	}
	b, err := json.Marshal(doc)
	if err != nil {
		return template.JS("{}")
	}
	return template.JS(b) //nolint:gosec // json.Marshal HTML-escapes <,>,& — safe in <script>
}

func firstImage(arts []prArticle) string {
	for _, a := range arts {
		if a.ImageURL != nil && strings.TrimSpace(*a.ImageURL) != "" {
			return toHTTPS(*a.ImageURL)
		}
	}
	return ""
}

func toHTTPS(u string) string {
	if strings.HasPrefix(u, "http://") {
		return "https://" + strings.TrimPrefix(u, "http://")
	}
	return u
}

func truncate(s string, max int) string {
	s = strings.TrimSpace(s)
	if len([]rune(s)) <= max {
		return s
	}
	r := []rune(s)
	return strings.TrimSpace(string(r[:max])) + "…"
}
