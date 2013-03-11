package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"github.com/gorilla/mux"
	"github.com/garyburd/redigo/redis"
)

const (
	REDIS_HOST = ""
	REDIS_PORT = "6379"
	REDIS_PREFIX = "getshorty:"
)

type Url struct {
	id string
	destination string
}

func (this *Url) Save() error {
	c, err := redis.Dial("tcp", REDIS_HOST + ":" + REDIS_PORT)
	defer c.Close()
	if err != nil {
		return err
	}

	reply, err := c.Do("SET", REDIS_PREFIX + "url:" + this.id, this.id)

	if err == nil && reply != "OK" {
		err = errors.New("Invalid Redis response")
	}

	if err != nil {
		return err
	}

	return nil
}

func GetUrl(id string) (*Url, error) {
	c, err := redis.Dial("tcp", REDIS_HOST + ":" + REDIS_PORT)
	if err != nil {
		return nil, err
	}

	defer c.Close()

	reply, err := c.Do("GET", REDIS_PREFIX + "url:" + id)
	if reply == nil {
		return nil, nil
	}

	reply, err = redis.String(reply, err)
	if err != nil {
		return nil, err
	}

	fmt.Println("GET RESPONSE:")
	fmt.Println(reply)

	return &Url{id: id}, nil
}

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
	io.WriteString(response, "Go to " + url.id)
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
	render(response, "stats", map[string]string{ "id": url.id })
}

func HomeHandler(response http.ResponseWriter, request *http.Request) {
	render(response, "home", map[string]string{ "name": "Golang" })
}

func render(response http.ResponseWriter, view string, data interface{}) {
	v := LoadView(view)
	v.Set(data)

	body, error := v.Render()
	if error != nil {
		fmt.Println(error)
		return
	}
	response.Write(body)
}

func main() {
	/*
	u := &Url{id: "test01"}
	err := u.Save()
	if err != nil {
		fmt.Println("ERROR: ")
		fmt.Println(err)
	}
	*/

	router := mux.NewRouter()
	router.HandleFunc("/{id:[a-z0-9]{6}}", ViewHandler).Name("view")
	router.HandleFunc("/{id:[a-z0-9]{6}}/stats", StatsHandler).Name("stats")
	router.HandleFunc("/", HomeHandler)

	http.ListenAndServe(":8080", router)
}
