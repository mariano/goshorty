package main

import (
	"io"
	"net/http"
	"github.com/gorilla/mux"
)

var router = mux.NewRouter()

func ViewHandler(resp http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	url, err := GetUrl(vars["id"])
	if err != nil {
		RenderError(resp, err.Error(), http.StatusInternalServerError)
		return
	} else if url == nil {
		RenderError(resp, "No URL was found with that goshorty code", http.StatusNotFound)
		return
	}
	io.WriteString(resp, "Go to " + url.Id)
}

func StatsHandler(resp http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	url, err := GetUrl(vars["id"])
	if err != nil {
		RenderError(resp, err.Error(), http.StatusInternalServerError)
		return
	} else if url == nil {
		RenderError(resp, "No URL was found with that goshorty code", http.StatusNotFound)
		return
	}
	Render(resp, "stats", map[string]string{ "id": url.Id })
}

func HomeHandler(resp http.ResponseWriter, req *http.Request) {
	Render(resp, "home", map[string]string{ "name": "Golang" })
}

func main() {
	/*
	u := &Url{Id: "test01"}
	err := u.Save()
	if err != nil {
		fmt.Println("ERROR: ")
		fmt.Println(err)
	}

	u, err := GetUrl("test01")
	if err != nil {
		fmt.Println("ERROR: ")
		fmt.Println(err)
	}
	u.Delete()
	*/

	router.HandleFunc("/{id:[a-z0-9]{6}}", ViewHandler).Name("view")
	router.HandleFunc("/{id:[a-z0-9]{6}}/stats", StatsHandler).Name("stats")
	router.HandleFunc("/", HomeHandler).Name("home")
	for _, dir := range []string{"css", "js", "img"} {
		router.PathPrefix("/" + dir + "/").Handler(http.StripPrefix("/" + dir + "/", http.FileServer(http.Dir("assets/" + dir))))
	}

	http.ListenAndServe(":8080", router)
}
