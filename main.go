/*
 *   Copyright (c) 2024 Edward Stock

 *   This program is free software: you can redistribute it and/or modify
 *   it under the terms of the GNU General Public License as published by
 *   the Free Software Foundation, either version 3 of the License, or
 *   (at your option) any later version.

 *   This program is distributed in the hope that it will be useful,
 *   but WITHOUT ANY WARRANTY; without even the implied warranty of
 *   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *   GNU General Public License for more details.

 *   You should have received a copy of the GNU General Public License
 *   along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/houseme/mobiledetect"
	_ "github.com/mattn/go-sqlite3"
)

func viewHandler(w http.ResponseWriter, r *http.Request, title string, userAgent string) {
	//category := r.URL.Path[len("/title/"):]
	if len(r.URL.Path) >= len("/title/") && r.URL.Path[:len("/title/")] == "/title/" {
		// Path starts with "/title/" and has enough characters for slicing
		category := r.URL.Path[len("/title/"):]

		// Process the extracted category

		switch {
		//Uses the prefix Help: making navigating to Help easier
		case strings.HasPrefix(category, "Help:"):
			specialPageName := strings.TrimPrefix(category, "Help:")
			specialPageName = strings.TrimSpace(specialPageName)
			fmt.Println("Help page accessed:", specialPageName)

			p, err := loadPage("Help-"+specialPageName, userAgent)
			if err != nil {
				fmt.Println("Error Occured in:", specialPageName)
				http.Redirect(w, r, "/", http.StatusFound)
				return
			}
			renderTemplate(w, "title", p)
			//Gets a list of pages and Random lands on one
		case strings.HasPrefix(category, "Special:Random"):
			fmt.Println("Random page accessed")
			db, err := loadDatabase()
			if err != nil {
				panic(err) // Handle errors appropriately in production
			}
			defer db.Close()

			// Fetch a random page title from the Pages table
			var title string
			row := db.QueryRow("SELECT title FROM Pages ORDER BY RANDOM() LIMIT 1") // Select only the title
			err = row.Scan(&title)
			if err != nil {
				fmt.Println("Error occurred in Random Page:", err)
				//http.Redirect(w, r, "/edit/"+title, http.StatusFound)
				return
			}

			http.Redirect(w, r, "/title/"+title, http.StatusFound) // Redirect to the randomly selected page

			// Assuming you have a renderTemplate function for rendering the Page struct
			//renderTemplate(w, "page", page, safeMenu)
		// Allows getting to Special pages which are basically functional pages
		case strings.HasPrefix(category, "Special:"):
			specialPageName := strings.TrimPrefix(category, "Special:")
			specialPageName = strings.TrimSpace(specialPageName)
			fmt.Println("Special page accessed:", specialPageName)

			p, err := loadPageSpecial(title, specialPageName, userAgent)
			if err != nil {
				fmt.Println("Error Occured in:", specialPageName)
				//http.Redirect(w, r, "/edit/"+title, http.StatusFound)
				return
			}
			renderTemplate(w, "title", p)
		//possibly for Editing Categories forgotten
		case strings.Contains(category, ":"):
			categoryParts := strings.Split(category, ":")
			categoryName := strings.TrimSpace(categoryParts[1])
			fmt.Println("Category: ", categoryName)
			p, err := loadPageCategory(title, categoryName, userAgent)
			if err != nil {
				fmt.Println("Error Occured in: Editing category")
				//http.Redirect(w, r, "/edit/"+title, http.StatusFound)
				return
			}
			renderTemplate(w, "title", p)
		// Normal View Normal View Normal View
		default:

			fmt.Println("No category specified, defaulting to normal view")

			// Load the page for standard title viewing
			p, err := loadPage(title, userAgent)
			if err != nil {
				fmt.Println("viewHandler: Something weird happened")
				http.Redirect(w, r, "/title/Main_page", http.StatusFound)
				return
			}

			renderTemplate(w, "title", p)

		}
	} else {
		//fmt.Println("hello beautiful world")
		// Load the page for standard title viewing
		p, err := loadPage("Main_page", userAgent)
		if err != nil {
			fmt.Println("viewHandler: Something weird happened")
			http.Redirect(w, r, "/title/Main_page", http.StatusFound)
			return
		}
		renderTemplate(w, "title", p)
	}

}

// Edit Handler with a switch for editing Categories
func editHandler(w http.ResponseWriter, r *http.Request, title string, userAgent string) {
	updated_at := "Not Available"
	fmt.Println(title)
	size := ""
	if userAgent == "desktop" {
		size = "<div class=\"col-11 d-none d-sm-block\">"
	} else {
		size = "<div class=\"col-12 d-block d-sm-none\">"
	}
	category := r.URL.Path[len("/title/"):]

	switch {
	case strings.Contains(category, ":"):
		categoryParts := strings.Split(category, ":")
		categoryName := strings.TrimSpace(categoryParts[1])
		fmt.Println("Category:", categoryName)

		session, _ := store.Get(r, "cookie-name")
		auth, ok := session.Values["authenticated"].(bool)

		if !ok || !auth {
			//http.Error(w, "Forbidden", http.StatusForbidden)
			http.Redirect(w, r, "/error", http.StatusFound)
			return
		} else {

			ep, err := loadCategoryNoHtml(categoryName, userAgent)

			if err != nil {
				ep = &EditPage{CTitle: categoryName, Title: categoryName, Size: template.HTML(size), UpdatedDate: updated_at}
			}
			renderEditPageTemplate(w, "editCategory", ep)
		}
	default:

		// check our user is logged in
		session, _ := store.Get(r, "cookie-name")
		auth, ok := session.Values["authenticated"].(bool)

		if !ok || !auth {
			//http.Error(w, "Forbidden", http.StatusForbidden)
			http.Redirect(w, r, "/error", http.StatusFound)
			return
		} else {

			ep, err := loadPageNoHtml(title, userAgent)

			if err != nil {
				ep = &EditPage{CTitle: removeUnderscores(title), Title: title, Size: template.HTML(size)}
			}
			renderEditPageTemplate(w, "edit", ep)
		}
	}
}

func saveCatHandler(w http.ResponseWriter, r *http.Request, title string, userAgent string) {
	body := r.FormValue("body")

	p := &Page{Title: title, Body: template.HTML(body)}
	err := p.saveCat()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/title/Special:Categories", http.StatusFound)
}

// var validPath = regexp.MustCompile("^/(?:(add|addpage|cat|edit|save|title|Category|Special)/([a-zA-Z0-9]+)|)")
// var validPath = regexp.MustCompile("^/(?:(add|addpage|cat|edit|save|title|Category|Special)/([a-zA-Z0-9_-]+)|)")
var validPath = regexp.MustCompile("^/(?:(admin|add|addpage|edit|delete|savecat|save|title|login|Category|Special)/([a-zA-Z0-9'_-]+)|)")

//var validPath = regexp.MustCompile("^/(edit|save|view)/([a-zA-Z0-9]+)$")

func makeHandler(fn func(http.ResponseWriter, *http.Request, string, string)) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		//requestHeaders := r.Header
		detect := mobiledetect.New(r, nil)
		userAgent := ""
		if detect.IsMobile() || detect.IsTablet() {
			fmt.Println("is either a mobile or tablet")
			userAgent = "mobile"
		} else {
			userAgent = "desktop"
		}

		//userAgent := requestHeaders.Get("User-Agent")
		//fmt.Println("User-Agent:", userAgent)

		m := validPath.FindStringSubmatch(r.URL.Path)

		if m == nil {
			http.NotFound(w, r)
			fmt.Println("something went wrong")
			return
		}

		fn(w, r, m[2], userAgent)
		//handleSSEUpdates(w, r)
	}
}
func deleteHandler(w http.ResponseWriter, r *http.Request) {
	// Extract the resource type and title using strings.SplitN
	parts := strings.SplitN(strings.TrimPrefix(r.URL.Path, "/delete/"), "/", 2)
	if len(parts) != 2 {
		http.Error(w, "Invalid URL format", http.StatusBadRequest)
		return
	}
	resourceType := parts[0]
	title := parts[1]

	// Handle deletion based on resource type
	if resourceType == "page" {
		// Handle deletion of a page
		p := &Page{Title: title}
		err := p.deletePage()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/admin/manage", http.StatusFound)
	} else if resourceType == "category" {
		// Handle deletion of a category
		cat := &Category{Title: title}
		err := cat.deleteCategory()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/admin/manage", http.StatusFound)
	} else {
		// Handle invalid resource type
		//http.Error(w, "Invalid resource type", http.StatusBadRequest)
		http.Redirect(w, r, "/error", http.StatusFound)
	}
}

func addHandler(w http.ResponseWriter, r *http.Request) {
	// check our user is logged in

	detect := mobiledetect.New(r, nil)
	size := ""
	if detect.IsMobile() || detect.IsTablet() {
		fmt.Println("is either a mobile or tablet")

		size = "<div class=\"col-12 d-block d-sm-none\">"
	} else {
		size = "<div class=\"col-11 d-none d-sm-block\">"
	}

	session, _ := store.Get(r, "cookie-name")
	auth, ok := session.Values["authenticated"].(bool)

	if !ok || !auth {
		http.Redirect(w, r, "/error", http.StatusFound)
		//http.Error(w, "Forbidden", http.StatusForbidden)
		return
	} else {

		title := ""
		safeMenu, err := loadMenu()
		if err != nil {
			log.Println("Error loading menu:", err)
			// Handle the error, e.g., display a user-friendly message
			return
		}
		// Create an AddPage instance directly (no loading from file)
		ap := &AddPage{NavTitle: config.SiteTitle, CTitle: "Add Page", Title: title, Menu: safeMenu, Size: template.HTML(size), UpdatedDate: ""}

		// Populate other fields of ap as needed (e.g., from session data, user input, etc.)

		renderAddPageTemplate(w, "add", ap)
	}
}

// Error page needs to be used
func errorPage(w http.ResponseWriter, r *http.Request) {
	detect := mobiledetect.New(r, nil)
	userAgent := ""
	if detect.IsMobile() || detect.IsTablet() {
		fmt.Println("is either a mobile or tablet")
		userAgent = "mobile"
	} else {
		userAgent = "desktop"
	}
	p, err := loadPageSpecial("Error", "specialPageName", userAgent)
	if err != nil {
		http.Error(w, "Error loading HTML file", http.StatusInternalServerError)
		return
	}
	renderTemplate(w, "errorPage", p)
	// htmlContent, err := os.ReadFile("templates/errorPage.html")

	// w.Header().Set("Content-Type", "text/html")
	// w.Write(htmlContent)
}
func dbsql(stater string, args ...interface{}) error {
	db, err := loadDatabase()
	if err != nil {
		fmt.Println("Database Error: " + err.Error())

	}
	defer db.Close() // Ensure database closure

	stmt, err := db.Prepare(stater)
	if err != nil {
		fmt.Println("Database Error: " + err.Error())
	}
	defer stmt.Close() // Close the prepared statement

	_, err = stmt.Exec(args...) // Execute the statement with provided arguments
	if err != nil {
		fmt.Println("Database Error: " + err.Error())
	}

	return nil // Indicate successful execution
}

func styleHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "images/lector.css")
}
func logoHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "images/arcwiki.svg")
}
func mdeHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "images/mdemod.js")
}
func validatorHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "images/validator.js")
}
func awbHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "images/arcwikibanner.png")
}

func gplLogoHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "images/gpl3.svg")
}

type Config struct {
	SiteTitle string `json:"siteTitle"`
}

var config Config // Package-level variable
func main() {
	dbSetup()
	config.SiteTitle = os.Getenv("SITENAME")
	if config.SiteTitle == "" {
		data, err := ioutil.ReadFile("config.json")
		if err != nil {
			panic(err) // Handle the error appropriately in production
		}

		// Unmarshal the JSON data

		err = json.Unmarshal(data, &config)
		if err != nil {
			panic(err) // Handle the error appropriately in production
		}

	}

	// Access the extracted value
	fmt.Println("Site Title:", config.SiteTitle)
	go func() {
		for {
			if err := updateCategoryLinks(); err != nil {
				// Handle error
				fmt.Println("Error updating categories:", err)
			}
			time.Sleep(20 * time.Second)
			if err := updateSubCategoryLinks(); err != nil {
				// Handle error
				fmt.Println("Error updating subcategories:", err)
			}
			time.Sleep(20 * time.Second)
		}
	}()

	http.HandleFunc("/", makeHandler(viewHandler))
	http.HandleFunc("/add", addHandler)
	http.HandleFunc("/addpage", addPage)
	http.HandleFunc("/delete/", deleteHandler)
	http.HandleFunc("/category/", addCat)
	http.HandleFunc("/savecat/", makeHandler(saveCatHandler))
	http.HandleFunc("/admin/", makeHandler(adminHandler))
	http.HandleFunc("/logout", makeHandler(logout))
	http.HandleFunc("/login", makeHandler(loginFormHandler))
	http.HandleFunc("/loginPost", makeHandler(loginHandler))
	http.HandleFunc("/title/", makeHandler(viewHandler))
	http.HandleFunc("/edit/", makeHandler(editHandler))
	http.HandleFunc("/save/", makeHandler(saveHandler))
	http.HandleFunc("/error", errorPage)
	http.HandleFunc("/images/lector.css", styleHandler)
	http.HandleFunc("/images/arcwiki.svg", logoHandler)
	http.HandleFunc("/images/arcwikibanner.png", awbHandler)
	http.HandleFunc("/images/gpl3.svg", gplLogoHandler)
	http.HandleFunc("/images/mdemod.js", mdeHandler)
	http.HandleFunc("/images/validator.js", validatorHandler)

	log.Fatal(http.ListenAndServe(":8080", nil))

}
