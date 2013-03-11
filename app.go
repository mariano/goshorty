package main

import (
	"fmt"
	"io"
	"net/http"
	"github.com/gorilla/mux"
)

func ViewHandler(response http.ResponseWriter, request *http.Request) {
	vars := mux.Vars(request)
	url, err := GetUrl(vars["id"])
	if err != nil {
		io.WriteString(response, "ERROR")
		fmt.Println(err)
		return
	} else if url == nil {
		io.WriteString(response, "NOT FOUND!")
		return
	}
	io.WriteString(response, "Go to " + url.Id)
}

func StatsHandler(response http.ResponseWriter, request *http.Request) {
	vars := mux.Vars(request)
	url, err := GetUrl(vars["id"])
	if err != nil {
		io.WriteString(response, "ERROR")
		fmt.Println(err)
		return
	} else if url == nil {
		io.WriteString(response, "NOT FOUND!")
		return
	}
	RenderView(response, "stats", map[string]string{ "id": url.Id })
}

func HomeHandler(response http.ResponseWriter, request *http.Request) {
	RenderView(response, "home", map[string]string{ "name": "Golang" })
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

	router := mux.NewRouter()
	router.HandleFunc("/{id:[a-z0-9]{6}}", ViewHandler).Name("view")
	router.HandleFunc("/{id:[a-z0-9]{6}}/stats", StatsHandler).Name("stats")
	router.HandleFunc("/", HomeHandler)

	http.ListenAndServe(":8080", router)
}
