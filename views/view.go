package views

import (
	"bytes"
	"errors"
	"github.com/gorilla/csrf"
	"html/template"
	"io"
	"net/http"
	"path/filepath"

	"github.com/cowboyhat-io/gittyupsec.com/context"
)

var (
	// LayoutDir is the directory containing
	// parts of the website that define
	// the whole
	LayoutDir string = "views/layouts/"
	// TemplateDir contains all of the templates to be rendered
	// as pages
	TemplateDir string = "views/"
	// TemplateExt is the ending of the template files
	TemplateExt string = ".gohtml"
)

// View contains data that is used to render
// webpage
type View struct {
	Template *template.Template
	Layout   string
}

func layoutFiles() []string {
	files, err := filepath.Glob(LayoutDir + "*" + TemplateExt)
	if err != nil {
		panic(err)
	}
	return files
}

func addTemplatePath(files []string) {
	for i, f := range files {
		files[i] = TemplateDir + f
	}
}

func addTemplateExt(files []string) {
	for i, f := range files {
		files[i] = f + TemplateExt
	}
}

// NewView is used to parse all of the templates
// and return a view
func NewView(layout string, files ...string) *View {
	addTemplatePath(files)
	addTemplateExt(files)
	files = append(files, layoutFiles()...)
	// We are changing how we create our templates, calling
	// New("") to give us a template that we can add a function to
	// before finally passing in files to parse as part of the template.
	t, err := template.New("").Funcs(template.FuncMap{
		"csrfField": func() (template.HTML, error) {
			// If this is called without being replace with a proper implementation
			// returning an error as the second argument will cause our template
			// package to return an error when executed.
			return "", errors.New("csrfField is not implemented")
		},
		// Once we have our template with a function we are going to pass in files
		// to parse, much like we were previously.
	}).ParseFiles(files...)
	if err != nil {
		panic(err)
	}

	return &View{
		Template: t,
		Layout:   layout,
	}
}

// Render accepts a ptr to a View and then executes
// the template(s), filling in the data, to build
// webpages
func (v *View) Render(w http.ResponseWriter, r *http.Request, data interface{}) {
	w.Header().Set("Content-Type", "text/html")
	var vd Data
	switch d := data.(type) {
	case Data:
		// We need to do this so we can access the data in a var
		// with the type Data.
		vd = d
	default:
		// If the data IS NOT of the type Data, we create one
		// and set the data to the Yield field like before.
		vd = Data{
			Yield: data,
		}
	}
	// Lookup and set the user to the User field
	vd.User = context.User(r.Context())
	var buf bytes.Buffer
	// We need to create the csrfField using the current http request.
	csrfField := csrf.TemplateField(r)
	tpl := v.Template.Funcs(template.FuncMap{
		// We can also change the return type of our function, since we no longer
		// need to worry about errors.
		"csrfField": func() template.HTML {
			// We can then create this closure that returns the csrfField for
			// any templates that need access to it.
			return csrfField
		},
	})
	// Then we continue to execute the template just like before.
	err := tpl.ExecuteTemplate(&buf, v.Layout, vd)
	if err != nil {
		http.Error(w, "Something went wrong. If the problem "+
			"persists, please email support@lenslocked.com",
			http.StatusInternalServerError)
		return
	}
	io.Copy(w, &buf)
}

// ServeHTTP accepts a ptr to a view and renders
// the view view writing it as http.ResponseWriter
func (v *View) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	v.Render(w, r, nil)
}
