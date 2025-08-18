package handlers

import (
	"database/sql"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"strconv"
	"time"

	"test/database"
	"test/middleware"
	"test/models"
)

func HomeFunction(w http.ResponseWriter, r *http.Request) {
	// Check if continuing as guest
	isGuest := r.URL.Query().Get("guest") == "true"

	var data models.HomeData

	if isGuest {
		data = models.HomeData{
			Username: "Guest",
			IsGuest:  true,
			Posts:    getAllPosts(nil), // Get all posts for guest
		}
	} else {
		// Check for authenticated user
		user := middleware.GetCurrentUser(r)
		if user == nil {
			http.Redirect(w, r, "/welcome", http.StatusSeeOther)
			return
		}

		// Get user creation date and stats
		var createdAt time.Time
		database.DB.QueryRow("SELECT created_at FROM users WHERE id = ?", user.ID).Scan(&createdAt)

		stats := getUserStats(user.ID)

		posts1 := getAllPosts(user)
		length := len(posts1)
		pages := 0
		if length%10 == 0 {
			pages = length / 10
		} else {
			pages = length/10 + 1
		}

		cpage := r.URL.Query().Get("currentpage")
		page := 0
		if cpage == "" {
			page = 1
		} else {
			page1, err := strconv.Atoi(cpage)
			if err != nil {
				page = 1
			}
			page = page1

		}
		//need to apply error handling
		myrange := (page - 1) * 10
		posts2 := posts1[myrange : myrange+10]
		data = models.HomeData{
			Username:       user.Username,
			CreatedAt:      createdAt.Format("January 2, 2006"),
			IsGuest:        false,
			Posts:          posts2, // Get all posts for authenticated user
			Stats:          stats,
			CurrentPage:    page,
			AvailablePages: pages,
		}
	}

	RenderHomePage(w, r, data)
}

func HomePageFunction(w http.ResponseWriter, r *http.Request) {
	HomeFunction(w, r) // Same as root handler
}

func getAllPosts(user *models.User) []models.Post {
	query := `SELECT p.id, p.title, p.content, u.username, cat.category_names as category_name, p.created_at,
					   COALESCE(like_count, 0) as like_count,
					   COALESCE(comment_count, 0) as comment_count
				FROM posts p
				JOIN users u ON p.user_id = u.id
				LEFT JOIN (
					SELECT pc.post_id, GROUP_CONCAT(c.name, ', ') AS category_names
					FROM post_categories pc
					JOIN categories c ON c.id = pc.category_id
					GROUP BY pc.post_id
				) cat ON cat.post_id = p.id
				LEFT JOIN (
					SELECT post_id, COUNT(*) as like_count 
					FROM likes 
					WHERE post_id IS NOT NULL 
					GROUP BY post_id
				) l ON p.id = l.post_id
				LEFT JOIN (
					SELECT post_id, COUNT(*) as comment_count 
					FROM comments 
					GROUP BY post_id
				) cm ON p.id = cm.post_id
				ORDER BY p.created_at DESC LIMIT 20`

	rows, err := database.DB.Query(query)
	if err != nil {
		return []models.Post{}
	}
	defer rows.Close()

	var posts []models.Post
	for rows.Next() {
		var post models.Post
		var categoryName sql.NullString
		err := rows.Scan(&post.ID, &post.Title, &post.Content, &post.Username,
			&categoryName, &post.CreatedAt, &post.LikeCount, &post.CommentCount)
		if err != nil {
			continue
		}

		if categoryName.Valid {
			post.CategoryName = categoryName.String
		} else {
			post.CategoryName = "General"
		}

		// Check if user liked this post
		if user != nil {
			var liked int
			database.DB.QueryRow("SELECT COUNT(*) FROM likes WHERE user_id = ? AND post_id = ?",
				user.ID, post.ID).Scan(&liked)
			post.IsLiked = liked > 0
		}

		posts = append(posts, post)
	}

	return posts
}

func getUserStats(userID int) models.UserStats {
	var stats models.UserStats

	// Get post count
	database.DB.QueryRow("SELECT COUNT(*) FROM posts WHERE user_id = ?", userID).Scan(&stats.PostCount)

	// Get comment count
	database.DB.QueryRow("SELECT COUNT(*) FROM comments WHERE user_id = ?", userID).Scan(&stats.CommentCount)

	return stats
}

func RenderHomePage(w http.ResponseWriter, r *http.Request, data models.HomeData) {
	fmt.Printf("Attempting to load template from: templates/home.html\n")

	// Check if file exists and get file info
	fileInfo, err := os.Stat("templates/home.html")
	if err != nil {
		fmt.Printf("File stat error: %v\n", err)
		HandleError(w, r, 500, "Template file not found", "/home")
		return
	}

	fmt.Printf("File exists, size: %d bytes\n", fileInfo.Size())

	// Create template with functions and parse file
	tmpl, err := template.New("home.html").Funcs(template.FuncMap{
		"slice": func(s string, start, end int) string {
			if start < 0 || start >= len(s) {
				return ""
			}
			if end > len(s) {
				end = len(s)
			}
			if end <= start {
				return ""
			}
			return s[start:end]
		},
		"add": func(a, b int) int {
			return a + b
		},
	}).ParseFiles("templates/home.html")

	if err != nil {
		fmt.Printf("Template parse error: %v\n", err)
		HandleError(w, r, 500, fmt.Sprintf("Template parse error: %v", err), "/home")
		return
	}

	fmt.Printf("Template parsed successfully\n")

	w.Header().Set("Content-Type", "text/html")

	// Execute the template by its filename
	err = tmpl.ExecuteTemplate(w, "home.html", data)
	if err != nil {
		fmt.Printf("Template execution error: %v\n", err)
		fmt.Printf("Data being passed: %+v\n", data)
		return
	}

	fmt.Printf("Template executed successfully\n")
}
