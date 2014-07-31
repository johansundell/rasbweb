// raspweb project main.go
package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"html/template"
	"log"
	"net/http"
)

/*const (
	mysqlDb   = "rasbweb"
)*/
var mysqlUser, mysqlPass, mysqlDb string

type Page struct {
	Title string
	Body  string
	Image string
}

func loadPage(str string) *Page {
	p := &Page{"Title", str, ""}
	return p
}

func myHandler(w http.ResponseWriter, r *http.Request) {
	t, _ := template.ParseFiles("tmpl/index.html")
	log.Println(r.RequestURI)
	p := loadPage("test")
	t.Execute(w, p)
}

func myJson(w http.ResponseWriter, r *http.Request) {
	defer func() bool {
		if r := recover(); r != nil {
			log.Println("SUDDE: ", r)

			return false
		}
		return true
	}()

	db, sqlerr := sql.Open("mysql", mysqlUser+":"+mysqlPass+"@/"+mysqlDb)

	defer db.Close()
	//log.Println(r.Method)
	//log.Println(r.URL.RawQuery)
	r.ParseForm()
	log.Println(r.Form.Get("test"))
	cat := r.Form.Get("test")
	if sqlerr != nil {
		panic(sqlerr)
	}
	rows, err := db.Query("SELECT title, body FROM articles a JOIN categories c ON c.category_id = a.category_id WHERE c.alias = ?", cat)

	if err != nil {
		panic(err)
	}
	slice := []Page{}
	for rows.Next() {
		p := Page{}
		rows.Scan(&p.Title, &p.Body)
		slice = append(slice, p)
	}
	/*p1 := Page{"Title1", "Body1", "Image1"}
	p2 := Page{"Title2", "Body2", "Image2"}

	slice = append(slice, p1)
	slice = append(slice, p2)*/
	log.Println("here")
	b, err := json.MarshalIndent(slice, "", "    ")
	if err == nil {
		fmt.Fprintf(w, string(b))
	}
}

func main() {
	flag.StringVar(&mysqlUser, "dbuser", "", "The database user")
	flag.StringVar(&mysqlPass, "dbpass", "", "The database password")
	flag.StringVar(&mysqlDb, "db", "", "The database")

	flag.Parse()

	http.HandleFunc("/", myHandler)

	http.HandleFunc("/json", myJson)

	http.HandleFunc("/bower_components/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, r.URL.Path[1:])
	})

	http.HandleFunc("/elements/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, r.URL.Path[1:])
	})

	panic(http.ListenAndServe(":8080", nil))
}
