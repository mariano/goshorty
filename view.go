package main

import (
	"bytes"
	"html/template"
	"net/http"
	"path/filepath"
	"sync"
)

type registry struct {
	sync.RWMutex
	templates map[string]*template.Template
}

var r registry

func init() {
	r.templates = make(map[string]*template.Template)
}

func Render(resp http.ResponseWriter, req *http.Request, view string, data interface{}) (err error) {
	body, err := render(req, "layout", view, data)
	if err != nil {
		http.Error(resp, err.Error(), http.StatusInternalServerError)
		return
	}
	resp.Write(body)
	return
}

func RenderError(resp http.ResponseWriter, req *http.Request, message string, code int) (err error) {
	body, err := render(req, "layout", "error", map[string]string{"Error": message})
	if err != nil {
		http.Error(resp, err.Error(), http.StatusInternalServerError)
		return
	}

	resp.WriteHeader(code)
	resp.Write(body)
	return
}

func render(req *http.Request, layout string, name string, data interface{}) (body []byte, err error) {
	view, err := parse(req, "views/"+name+".html", data)
	if err != nil {
		return
	}
	body, err = parse(req, "views/layouts/"+layout+".html", map[string]template.HTML{"Content": template.HTML(view)})
	if err != nil {
		return
	}
	return
}

func parse(req *http.Request, file string, data interface{}) (body []byte, err error) {
	r.RLock()
	t, present := r.templates[file]
	r.RUnlock()

	if !present {
		buildURL := func(name string, full bool, args ...string) (urlString string) {
			route := router.Get(name)
			if route == nil {
				return ""
			}

			url, err := route.URL(args...)
			if err != nil {
				return ""
			}
			urlString = url.String()
			if full {
				if urlString[0:1] != "/" {
					urlString = "/" + urlString
				}
				urlString = "http://" + req.Host + urlString
			}
			return
		}
		t = template.New(filepath.Base(file))
		t.Funcs(template.FuncMap{
			"full_url": func(name string, args ...string) string {
				return buildURL(name, true, args...)
			},
			"url": func(name string, args ...string) string {
				return buildURL(name, false, args...)
			},
		})

		_, err = t.ParseFiles(file)
		if err != nil {
			return
		}
		r.Lock()
		r.templates[file] = t
		r.Unlock()
	}

	var buf bytes.Buffer
	err = t.Execute(&buf, data)
	if err != nil {
		return
	}
	return buf.Bytes(), nil
}
