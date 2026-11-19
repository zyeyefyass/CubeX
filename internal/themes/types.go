package themes

// ThemeMeta represents metadata and token definitions for a CubeX theme.
type ThemeMeta struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	Type        string            `json:"type"` // "vector", "multi-image", "video"
	Tier        string            `json:"tier"` // "builtin", "legacy", "free-fan", "member"
	Description string            `json:"description"`
	Accent      string            `json:"accent"`
	Assets      map[string]string `json:"assets"` // e.g. {"video": "bg.mp4", "poster": "poster.webp", "preview": "preview.webp"}
	Tokens      map[string]string `json:"tokens"` // e.g. {"--cbx-theme-accent": "#E5B95C"}
	IsBuiltin   bool              `json:"is_builtin"`
	IsActive    bool              `json:"is_active"`
}

// ThemeConfig represents persisted active theme state.
type ThemeConfig struct {
	ActiveThemeID string `json:"active_theme_id"`
	PlaybackMode  string `json:"playback_mode"` // "auto", "always", "poster"
}

// ThemesResponse is the payload returned by GET /api/themes.
type ThemesResponse struct {
	ActiveThemeID string      `json:"active_theme_id"`
	PlaybackMode  string      `json:"playback_mode"`
	Themes        []ThemeMeta `json:"themes"`
}
