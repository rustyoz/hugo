// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package commands

import (
	"flag"
	"github.com/spf13/hugo/helpers"
	"html/template"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	addr = flag.Bool("addr", false, "find open address and print to final-port.txt")
)

func createTemplates() *template.Template {
	temps, _ := template.New("edit").Parse(editTemplate())
	temps.New("view").Parse(viewTemplate())
	return temps
}

var templates = createTemplates()

type Page struct {
	Title string
	Body  []byte
	Url   string
}

func (p *Page) save() error {
	path := filepath.Join(helpers.GetContentDirPath(), filepath.FromSlash(p.Title))

	return ioutil.WriteFile(path, p.Body, 0600)
}

func loadPage(filename string) (*Page, error) {
	path := filepath.Join(helpers.GetContentDirPath(), filepath.FromSlash(filename))
	body, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return &Page{Title: filename, Body: body, Url: `/` + strings.TrimSuffix(filename, filepath.Ext(filename))}, nil
}

func viewHandler(w http.ResponseWriter, r *http.Request, title string) {
	p, err := loadPage(title)
	if err != nil {
		http.Redirect(w, r, "/edit/"+title, http.StatusFound)
		return
	}
	renderTemplate(w, "view", p)
}

func editHandler(w http.ResponseWriter, r *http.Request, title string) {
	p, err := loadPage(title)
	if err != nil {
		p = &Page{Title: title}
	}
	renderTemplate(w, "edit", p)
}

func saveHandler(w http.ResponseWriter, r *http.Request, title string) {
	body := r.FormValue("body")
	p := &Page{Title: title, Body: []byte(body)}
	err := p.save()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/view/"+title, http.StatusFound)
}

func directoryHandler(w http.ResponseWriter, r *http.Request, path string) {
	dir := helpers.GetContentDirPath() + filepath.FromSlash(path)
	http.ServeFile(w, r, dir)
}

func renderTemplate(w http.ResponseWriter, tmpl string, p *Page) {
	err := templates.ExecuteTemplate(w, tmpl, p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

var validFilePath = regexp.MustCompile(`^/(edit/|save/|view/)([a-zA-Z0-9]+/)*([a-zA-Z0-9]+\.[a-zA-Z0-9]{1,3})$`)
var validDirPath = regexp.MustCompile(`^(/view)(/[a-zA-Z0-9]*/?)*$`)

func makeHandler(fn func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		f := validFilePath.FindStringSubmatch(r.URL.Path)
		d := validDirPath.FindStringSubmatch(r.URL.Path)

		if f != nil {
			fn(w, r, f[2]+f[3])
		} else if d != nil {
			directoryHandler(w, r, d[2])
		} else {
			http.NotFound(w, r)
			return
		}

	}
}

func editTemplate() string {
	return `
<script src="http://ajax.googleapis.com/ajax/libs/jquery/1.4.3/jquery.min.js"></script>

<script src="/epiceditor/js/epiceditor.js"></script>

<div style="width: 50%; float: left; height: 100%; display: flex; flex-direction: column; flex-grow: 1;" >
<h1>Editing {{.Title}}</h1>
<form id="edit" action="/save/{{.Title}}" method="POST" style="display: flex; flex-direction: column; flex-grow: 1;" >
<textarea  id="body" name="body" style="display: none;">{{printf "%s" .Body}}</textarea>
<div id="epiceditor" style="flex-grow: 1;"></div>
<input type="submit" value="Save">
</form>
</div>
</div>
<div style="width: 50%; float: right; height: 100%; display: flex; flex-direction: column;">
<iframe src="{{ .Url }}" style="width: 100%; flex-grow: 1;"></iframe>
</div>

<script>
// Attach a submit handler to the form
$( "#edit" ).submit(function( event ) {
 
 var $form = $(this);
  // Stop form from submitting normally
  event.preventDefault();

  var postdata = $form.serialize();
  var posturl = $form.attr( "action" );

  $.ajax({
	url: posturl,
	type: "post", 
	data: postdata
  });


});
</script>
<script>
var opts ={ basePath: '/epiceditor', textarea: "body",}
var editor = new EpicEditor(opts).load();</script>

`

}

func viewTemplate() string {
	return `
<div style="width: 50%; float: left;">
<h1>{{.Title}}</h1>
<p>[<a href="/edit/{{.Title}}">edit</a>]</p>
<div>{{printf "%s" .Body}}</div>
</div>
<div style="width: 50%; float: right;">
<iframe src="{{ .Url }}" style="width: 100%; height: 100%"></iframe>
</div>
`
}
