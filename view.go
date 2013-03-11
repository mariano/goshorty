package main

import (
	"bytes"
	"html/template"
)

var templates = make(map[string]*template.Template)

func LoadView(name string) *View {
	v := &View{name: name, layout: "layout"}
	return v
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
		t, err := template.ParseFiles(file)
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
