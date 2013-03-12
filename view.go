package main

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
)

var templates = make(map[string]*template.Template)

func RenderView(response http.ResponseWriter, view string, data interface{}) {
	v := &View{name: view, layout: "layout"}
	v.Set(data)

	body, error := v.Render()
	if error != nil {
		fmt.Println(error)
		return
	}
	response.Write(body)
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

func (this *View) Render() (body []byte, error error) {
	return this.build("views/" + this.name + ".html", this.data)
}

func (this *View) build(file string, data interface{}) (body []byte, error error) {
	view, error := this.parse(file, data)
	if error != nil {
		return nil, error
	}
	layout, error := this.parse("views/layouts/" + this.layout + ".html", content{View: this, Content: template.HTML(view)})
	if error != nil {
		return nil, error
	}
	return layout, nil
}

func (this *View) parse(file string, data interface{}) (body []byte, error error) {
	var buf bytes.Buffer
	if templates[file] == nil {
		t := template.New(filepath.Base(file))
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

		_, err := t.ParseFiles(file)
		if err != nil {
			return nil, err
		}
		templates[file] = t
	}

	err := templates[file].Execute(&buf, data)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
