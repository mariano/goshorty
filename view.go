package main

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
	"sync"
)

type registry struct {
	sync.RWMutex
	templates map[string]*template.Template
	js        map[string]string
}

var r registry

func init() {
	r.templates = make(map[string]*template.Template)
	r.js = make(map[string]string)
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

func RenderJsonError(resp http.ResponseWriter, req *http.Request, message string, code int) (err error) {
	resp.WriteHeader(code)
	resp.Write([]byte(fmt.Sprintf("{\"error\":\"%s\"}", message)))
	return
}

func render(req *http.Request, layout string, name string, data interface{}) (body []byte, err error) {
	file := "views/" + name + ".html"
	view, err := parse(req, file, data)
	if err != nil {
		return
	}

	r.RLock()
	javascript, _ := r.js[file]
	r.RUnlock()

	body, err = parse(req, "views/layouts/"+layout+".html", map[string]template.HTML{
		"Content":    template.HTML(view),
		"Javascript": template.HTML(javascript),
	})
	if err != nil {
		return
	}
	return
}

func parse(req *http.Request, file string, data interface{}) (body []byte, err error) {
	r.RLock()
	t, present := r.templates[file]
	r.RUnlock()

	var jsFiles []string

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
			"load_js": func(file string, args ...string) string {
				if file != "" {
					jsFiles = append(jsFiles, file)
				}
				return ""
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

	if !present && len(jsFiles) > 0 {
		javascript := ""
		for i, file := range jsFiles {
			if i > 0 {
				javascript += "\n"
			}
			javascript += fmt.Sprintf("<script type=\"text/javascript\" src=\"%s\"></script>", file)
		}

		r.Lock()
		r.js[file] = javascript
		r.Unlock()
	}

	return buf.Bytes(), nil
}
