package main

import (
	"bytes"
	"html/template"
	"net/http"
	"path/filepath"
)

var templates = make(map[string]*template.Template)

func Render(resp http.ResponseWriter, view string, data interface{}) (err error) {
	v := &View{name: view, layout: "layout"}
	v.Set(data)

	body, err := v.Render()
	if err != nil {
		http.Error(resp, err.Error(), http.StatusInternalServerError)
		return
	}
	resp.Write(body)
	return
}

func RenderError(resp http.ResponseWriter, message string, code int) (err error) {
	v := &View{name: "error", layout: "layout"}
	v.Set(map[string]string{ "Error": message })

	body, err := v.Render()
	if err != nil {
		http.Error(resp, err.Error(), http.StatusInternalServerError)
		return
	}

	resp.WriteHeader(code)
	resp.Write(body)
	return
}

type View struct {
	name string
	layout string
	data interface{}
}

type content struct {
	View *View
	Content template.HTML
}

func (this *View) Set(data interface{}) {
	this.data = data
}

func (this *View) Render() (body []byte, err error) {
	return this.build("views/" + this.name + ".html", this.data)
}

func (this *View) build(file string, data interface{}) (body []byte, err error) {
	view, err := this.parse(file, data)
	if err != nil {
		return
	}
	body, err = this.parse("views/layouts/" + this.layout + ".html", content{View: this, Content: template.HTML(view)})
	if err != nil {
		return
	}
	return
}

func (this *View) parse(file string, data interface{}) (body []byte, err error) {
	var buf bytes.Buffer
	t, present := templates[file]
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
		templates[file] = t
	}

	err = t.Execute(&buf, data)
	if err != nil {
		return
	}
	return buf.Bytes(), nil
}
