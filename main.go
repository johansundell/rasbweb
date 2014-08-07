// raspweb project main.go
package main

import (
	"compress/gzip"
	//"compress/zlib"
	"database/sql"
	"encoding/json"
	"flag"
	//"fmt"
	_ "github.com/go-sql-driver/mysql"
	"html/template"
	"log"
	"net/http"
	"os/exec"
	"strings"
)

/*const (
	mysqlDb   = "rasbweb"
)*/
var mysqlUser, mysqlPass, mysqlDb, uuid string
var uses_gzip *bool

const fs_maxbufsize = 4096

type Page struct {
	Title string
	Body  string
	Image string
	Uuid  string
}

func loadPage(str string) *Page {
	p := &Page{"Title", str, "", uuid}
	return p
}

func myHandler(w http.ResponseWriter, r *http.Request) {
	if r.RequestURI == "/" || r.RequestURI == "index.html" {
		t, _ := template.ParseFiles("tmpl/index.html")
		w.Header().Set("Content-Type", "text/html")
		w.Header().Set("Content-Encoding", "gzip")
		log.Println("Page: " + r.RequestURI)
		p := loadPage("test")
		//log.Println(p.Uuid)

		gz := gzip.NewWriter(w)
		defer gz.Close()

		t.Execute(gz, p)
	} else {
		http.Error(w, "404 Not Found : Error while opening the file.", 404)
	}
	//t.Execute(w, p)

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
	//b, err := json.MarshalIndent(slice, "", "    ")
	if err == nil {
		//fmt.Fprintf(w, string(b))

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Encoding", "gzip")
		gz := gzip.NewWriter(w)
		defer gz.Close()
		json.NewEncoder(gz).Encode(slice)
		//gz.Close()
		//json.NewEncoder(w).Encode(slice)
	}
}

func main() {
	flag.StringVar(&mysqlUser, "dbuser", "johan", "The database user")
	flag.StringVar(&mysqlPass, "dbpass", "", "The database password")
	flag.StringVar(&mysqlDb, "db", "rasbweb", "The database")
	uses_gzip = flag.Bool("gzip", true, "Enables gzip/zlib compression")
	bind := flag.String("bind", ":8080", "Bind address")

	flag.Parse()

	uuid = getUuid()
	log.Println("UUID: " + uuid)

	http.HandleFunc("/", myHandler)

	http.HandleFunc("/json", myJson)

	http.HandleFunc("/"+uuid+"/bower_components/", func(w http.ResponseWriter, r *http.Request) {
		//http.ServeFile(w, r, r.URL.Path[1:])
		//log.Println(http.StripPrefix("bower_components", r.URL.Path[1:]))
		prefix := uuid
		if p := strings.TrimPrefix(r.URL.Path[1:], prefix); len(p) < len(r.URL.Path[1:]) {
			//log.Println(p[1:])
			serveFile(p[1:], w, r, true)
		} else {
			//serveFile(r.URL.Path[1:], w, r)
			http.Error(w, "404 Not Found : Error while opening the file.", 404)
		}
	})

	http.HandleFunc("/elements/", func(w http.ResponseWriter, r *http.Request) {
		//http.ServeFile(w, r, r.URL.Path[1:])
		serveFile(r.URL.Path[1:], w, r, false)
	})

	panic(http.ListenAndServe((*bind), nil))
}

func getUuid() string {
	out, err := exec.Command("uuidgen").Output()
	if err != nil {
		log.Fatal("Kunde inte hitta uuidgen")
	}
	return strings.TrimSpace(string(out))
}
