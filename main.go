// raspweb project main.go
package main

import (
	"compress/gzip"
	"compress/zlib"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"html/template"
	"io"
	"log"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

/*const (
	mysqlDb   = "rasbweb"
)*/
var mysqlUser, mysqlPass, mysqlDb string
var uses_gzip *bool

const fs_maxbufsize = 4096

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
	w.Header().Set("Content-Type", "text/html")
	w.Header().Set("Content-Encoding", "gzip")
	log.Println(r.RequestURI)
	p := loadPage("test")

	gz := gzip.NewWriter(w)
	defer gz.Close()

	t.Execute(gz, p)
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
	flag.StringVar(&mysqlUser, "dbuser", "", "The database user")
	flag.StringVar(&mysqlPass, "dbpass", "", "The database password")
	flag.StringVar(&mysqlDb, "db", "rasbweb", "The database")
	uses_gzip = flag.Bool("gzip", true, "Enables gzip/zlib compression")
	bind := flag.String("bind", ":8080", "Bind address")

	flag.Parse()

	http.HandleFunc("/", myHandler)

	http.HandleFunc("/json", myJson)

	http.HandleFunc("/bower_components/", func(w http.ResponseWriter, r *http.Request) {
		//http.ServeFile(w, r, r.URL.Path[1:])
		serveFile(r.URL.Path[1:], w, r)
	})

	http.HandleFunc("/elements/", func(w http.ResponseWriter, r *http.Request) {
		//http.ServeFile(w, r, r.URL.Path[1:])
		serveFile(r.URL.Path[1:], w, r)
	})

	panic(http.ListenAndServe((*bind), nil))
}

func serveFile(filepath string, w http.ResponseWriter, req *http.Request) {
	f, err := os.Open(filepath)

	if err != nil {
		http.Error(w, "404 Not Found : Error while opening the file.", 404)
		return
	}

	defer f.Close()

	// Checking if the opened handle is really a file
	statinfo, err := f.Stat()
	if err != nil {
		http.Error(w, "500 Internal Error : stat() failure.", 500)
		return
	}

	if statinfo.IsDir() {
		http.Error(w, "500 Internal Error : dir() failure.", 500)
		return
	}

	if (statinfo.Mode() &^ 07777) == os.ModeSocket { // If it's a socket, forbid it !
		http.Error(w, "403 Forbidden : you can't access this resource.", 403)
		return
	}

	// Manages If-Modified-Since and add Last-Modified (taken from Golang code)
	if t, err := time.Parse(http.TimeFormat, req.Header.Get("If-Modified-Since")); err == nil && statinfo.ModTime().Unix() <= t.Unix() {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Header().Set("Last-Modified", statinfo.ModTime().Format(http.TimeFormat))

	// Content-Type handling
	query, err := url.ParseQuery(req.URL.RawQuery)

	if err == nil && len(query["dl"]) > 0 { // The user explicitedly wanted to download the file (Dropbox style!)
		w.Header().Set("Content-Type", "application/octet-stream")
	} else {
		// Fetching file's mimetype and giving it to the browser
		if mimetype := mime.TypeByExtension(path.Ext(filepath)); mimetype != "" {
			w.Header().Set("Content-Type", mimetype)
		} else {
			w.Header().Set("Content-Type", "application/octet-stream")
		}
	}

	// Manage Content-Range (TODO: Manage end byte and multiple Content-Range)
	if req.Header.Get("Range") != "" {
		start_byte := parseRange(req.Header.Get("Range"))

		if start_byte < statinfo.Size() {
			f.Seek(start_byte, 0)
		} else {
			start_byte = 0
		}

		w.Header().Set("Content-Range",
			fmt.Sprintf("bytes %d-%d/%d", start_byte, statinfo.Size()-1, statinfo.Size()))
	}

	// Manage gzip/zlib compression
	output_writer := w.(io.Writer)

	is_compressed_reply := false

	if (*uses_gzip) == true && req.Header.Get("Accept-Encoding") != "" {
		encodings := parseCSV(req.Header.Get("Accept-Encoding"))

		for _, val := range encodings {
			if val == "gzip" {
				w.Header().Set("Content-Encoding", "gzip")
				output_writer = gzip.NewWriter(w)

				is_compressed_reply = true

				break
			} else if val == "deflate" {
				w.Header().Set("Content-Encoding", "deflate")
				output_writer = zlib.NewWriter(w)

				is_compressed_reply = true

				break
			}
		}
	}

	if !is_compressed_reply {
		// Add Content-Length
		w.Header().Set("Content-Length", strconv.FormatInt(statinfo.Size(), 10))
	}

	// Stream data out !
	buf := make([]byte, min(fs_maxbufsize, statinfo.Size()))
	n := 0
	for err == nil {
		n, err = f.Read(buf)
		output_writer.Write(buf[0:n])
	}

	// Closes current compressors
	switch output_writer.(type) {
	case *gzip.Writer:
		output_writer.(*gzip.Writer).Close()
	case *zlib.Writer:
		output_writer.(*zlib.Writer).Close()
	}

	f.Close()
}

func min(x int64, y int64) int64 {
	if x < y {
		return x
	}
	return y
}

func parseRange(data string) int64 {
	stop := (int64)(0)
	part := 0
	for i := 0; i < len(data) && part < 2; i = i + 1 {
		if part == 0 { // part = 0 <=> equal isn't met.
			if data[i] == '=' {
				part = 1
			}

			continue
		}

		if part == 1 { // part = 1 <=> we've met the equal, parse beginning
			if data[i] == ',' || data[i] == '-' {
				part = 2 // part = 2 <=> OK DUDE.
			} else {
				if 48 <= data[i] && data[i] <= 57 { // If it's a digit ...
					// ... convert the char to integer and add it!
					stop = (stop * 10) + (((int64)(data[i])) - 48)
				} else {
					part = 2 // Parsing error! No error needed : 0 = from start.
				}
			}
		}
	}

	return stop
}

func parseCSV(data string) []string {
	splitted := strings.SplitN(data, ",", -1)

	data_tmp := make([]string, len(splitted))

	for i, val := range splitted {
		data_tmp[i] = strings.TrimSpace(val)
	}

	return data_tmp
}
