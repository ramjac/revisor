package main

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
)

var (
	PathList  map[string]bool
	PathRoot  string
	templates map[string]*template.Template
)

type Page struct {
	Title     string
	Body      []byte
	Directory *map[string]bool
}

//this is a bit more scalable than we need since we only have one layout
func initTemplates() {
	if templates == nil {
		templates = make(map[string]*template.Template)
	}

	//think this'll get hard coded at first
	//config.Templates.Path
	templatesDir := "./"

	//implement Rob's error handling advice on these errors
	layouts, err := filepath.Glob(templatesDir + "layouts/*.tmpl")
	if err != nil {
		fmt.Println("layout could not be loaded")
	}
	includes, err := filepath.Glob(templatesDir + "includes/*.tmpl")
	if err != nil {
		fmt.Println("include could not be loaded")
	}

	fmt.Println("Layouts: ", len(layouts))
	fmt.Println("Includes: ", len(includes))

	// Generate our templates map from our layouts/ and includes/ directories
	for _, layout := range layouts {
		files := append(includes, layout)
		templates[filepath.Base(layout)] = template.Must(template.ParseFiles(files...))
		fmt.Println("Found layout: ", layout)
	}
	for k, v := range templates {

		fmt.Println("Found template: ", k)
		fmt.Println("val: ", v)

	}
}

func (p *Page) save() error {
	filename := p.Title // + ".txt"
	return ioutil.WriteFile(filename, p.Body, 0600)
}

func loadPage(title string) (*Page, error) {
	filename := title //+ ".txt"
	body, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return &Page{Title: title, Body: body}, nil
}

func editHandler(w http.ResponseWriter, r *http.Request) {
	//fmt.Fprintf(w, "Received: %s", r.URL.Path[1:])
	//I need to prevent navigation up a dir here with some regex
	fmt.Println("edit request")
	title := r.URL.Path[len("/edit/"):]
	p, err := loadPage(title)
	if err != nil {
		//fmt.Println("not found")
		p = &Page{Title: title}
	}
	//p.Directory = make([]string, len(PathList))
	//p.Directory = PathList[0:]
	p.Directory = &PathList

	renderTemplate(w, "index.tmpl", p)
}

func dirHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("directory request")
	p := Page{}
	p.Title = r.URL.Path[len("/dir/"):]
	p.Directory = &PathList

	renderTemplate(w, "dir.tmpl", &p)
}

func saveHandler(w http.ResponseWriter, r *http.Request) {
	title := r.URL.Path[len("/save/"):]
	body := r.FormValue("body")
	p := &Page{Title: title, Body: []byte(body)}
	p.save()
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("directory rebuild failed")
		}
		http.Redirect(w, r, "/edit/"+title, http.StatusFound)
	}()

	filepath.Walk(PathRoot, visit)
}

func visit(path string, f os.FileInfo, err error) error {
	if err != nil {
		fmt.Printf("File or directory traversal error: %v\n", err)
	}

	//fmt.Printf("Visited: %s | %v\n", path, f.IsDir())
	//Files will link to the edit handler, directories to a different handler
	PathList[path] = f.IsDir() //[PathIndex] = path
	return nil
}

func renderTemplate(w http.ResponseWriter, name string, p *Page) error {
	fmt.Println("rendering template")
	//check that the template exists
	tmpl, ok := templates[name]
	if !ok {
		fmt.Println("not tmpl found")
		return fmt.Errorf("No template by found by the name %s", name)
	}
	fmt.Println("tmpl: ", tmpl)
	//this was in the example but I'm not sure it is really necessary
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	//base our layout page
	tmpl.ExecuteTemplate(w, "base", p)

	return nil

	//the old way
	//t, _ := template.ParseFiles(tmpl + ".html")
	//t.Execute(w, p)
}

func main() {
	fmt.Println("So it goes")
	initTemplates()

	//should add configuration options to append to this path
	PathList = make(map[string]bool)
	PathRoot := "./website"
	err := filepath.Walk(PathRoot, visit)
	fmt.Printf("filepath.Walk() returned %v\n", err)

	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	http.HandleFunc("/edit/", editHandler)
	http.HandleFunc("/dir/", dirHandler)
	http.HandleFunc("/save/", saveHandler)
	http.ListenAndServe(":8080", nil)
}
