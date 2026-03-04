package resolver

type Metadata struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Channel  string `json:"channel"`
	Duration int    `json:"duration"`
	URL      string `json:"url"`
	Path     string `json:"path"`
}
