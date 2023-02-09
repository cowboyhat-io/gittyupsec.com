package controllers

import "github.com/cowboyhat-io/gittyupsec.com/views"

// Static contains the webpages
// that are not rendering dynamic content
type Static struct {
	Home      *views.View
	Pricing   *views.View
	Limit     *views.View
	Privacy   *views.View
	Business  *views.View
	Breaches  *views.View
	Automatic *views.View
	Team      *views.View
}

// NewStatic returns the Static webpages for
// the web app.
func NewStatic() *Static {
	return &Static{
		Home:      views.NewView("bootstrap", "static/home"),
		Pricing:   views.NewView("bootstrap", "static/pricing"),
		Limit:     views.NewView("bootstrap", "static/limit"),
		Privacy:   views.NewView("bootstrap", "static/privacy"),
		Business:  views.NewView("bootstrap", "static/business"),
		Breaches:  views.NewView("bootstrap", "static/github-data-breaches"),
		Automatic: views.NewView("bootstrap", "static/automatic"),
		Team:      views.NewView("bootstrap", "static/team"),
	}
}
