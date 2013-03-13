package main

import (
	"net/http"
	"github.com/gorilla/mux"
)

var router = mux.NewRouter()

func AddHandler(resp http.ResponseWriter, req *http.Request) {
	url, err := NewUrl(req.FormValue("url"))
	if err != nil {
		Render(resp, "home", map[string]string{ "error": err.Error() })
		return
	}

	statsUrl, err := router.Get("stats").URL("id", url.Id)
	if err != nil {
		RenderError(resp, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(resp, req, statsUrl.String(), http.StatusFound)
}

func RedirectHandler(resp http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	url, err := GetUrl(vars["id"])
	if err != nil {
		RenderError(resp, err.Error(), http.StatusInternalServerError)
		return
	} else if url == nil {
		RenderError(resp, "No URL was found with that goshorty code", http.StatusNotFound)
		return
	}
	http.Redirect(resp, req, url.Destination, http.StatusMovedPermanently)
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
	Render(resp, "stats", map[string]string{ 
		"id": url.Id,
		"url": url.Destination,
	})
}

func HomeHandler(resp http.ResponseWriter, req *http.Request) {
	Render(resp, "home", nil)
}

func main() {
	router.HandleFunc("/add", AddHandler).Methods("POST").Name("add")
	router.HandleFunc("/{id:[a-z0-9]{6}}+", StatsHandler).Name("stats")
	router.HandleFunc("/{id:[a-z0-9]{6}}", RedirectHandler).Name("redirect")
	router.HandleFunc("/", HomeHandler).Name("home")
	for _, dir := range []string{"css", "js", "img"} {
		router.PathPrefix("/" + dir + "/").Handler(http.StripPrefix("/" + dir + "/", http.FileServer(http.Dir("assets/" + dir))))
	}

	http.ListenAndServe(":8080", router)
}
