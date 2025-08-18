package models

type HomeData struct {
	Username  string
	CreatedAt string
	IsGuest   bool
	Posts     []Post
	Stats     UserStats
	CurrentPage int
	AvailablePages int
}

type PostsPageData struct {
	Posts      []Post
	Categories []Category
	IsGuest    bool
	Username   string
	Filter     string
}

type ErrorData struct {
	Code        int
	Message     string
	Description string
	Directory string
}
