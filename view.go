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
	r.RLock()
	r.templates = make(map[string]*template.Template)
	r.RUnlock()
}

func Render(resp http.ResponseWriter, view string, data interface{}) (err error) {
	body, err := render("layout", view, data)
	if err != nil {
		http.Error(resp, err.Error(), http.StatusInternalServerError)
		return
	}
	resp.Write(body)
	return
}

func RenderError(resp http.ResponseWriter, message string, code int) (err error) {
	body, err := render("layout", "error", map[string]string{ "Error": message })
	if err != nil {
		http.Error(resp, err.Error(), http.StatusInternalServerError)
		return
	}

	resp.WriteHeader(code)
	resp.Write(body)
	return
}

func render(layout string, name string, data interface{}) (body []byte, err error) {
	view, err := parse("views/" + name + ".html", data)
	if err != nil {
		return
	}
	body, err = parse("views/layouts/" + layout + ".html", map[string]template.HTML{"Content": template.HTML(view)})
	if err != nil {
		return
	}
	return
}

func parse(file string, data interface{}) (body []byte, err error) {
	r.RLock()
	t, present := r.templates[file]
	r.RUnlock()

	if !present {
		t = template.New(filepath.Base(file))
		t.Funcs(template.FuncMap{
			"url": func(name string, args ...string)(string) {
				route := router.Get(name)
				if route == nil {
					return ""
				}

				url, err := route.URL(args...)
				if err != nil {
					return ""
				}
				return url.String()
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
