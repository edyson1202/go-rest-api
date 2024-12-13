package games

type Page struct {
	Page int32
	Size int32
}

// Represents a game
type Game struct {
	Id          int        `json:"id"`
	ReleaseYear int        `json:"releaseYear"`
	Name        string     `json:"name"`
	Category    string     `json:"category"`
	Categories  []Category `json:"categories"`
}

// Represents individual categories
type Category struct {
	Name string `json:"name"`
}
