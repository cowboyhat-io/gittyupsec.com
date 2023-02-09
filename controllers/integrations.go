package controllers

import (
	C "context"
	"fmt"
	"github.com/cowboyhat-io/gittyupsec.com/context"
	"github.com/cowboyhat-io/gittyupsec.com/models"
	"github.com/cowboyhat-io/gittyupsec.com/views"
	"github.com/google/go-github/v50/github"
	"github.com/gorilla/mux"
	"golang.org/x/oauth2"
	"log"
	"net/http"
	"strconv"
	"strings"
)

const (
	IndexIntegrations = "index_integrations"
	EditIntegration   = "edit_integration"
)

type Repo struct {
	UserID      uint   `gorm:"not_null;index"`
	Name        string `gorm:"not_null"`
	Protections bool   `gorm:"not_null"`
	CQL         bool   `gorm:"not_null"`
	Secrets     bool   `gorm:"not_null"`
	Dependabot  bool   `gorm:"not_null"`
}

// NewIntegrations is for creating a new integration
func NewIntegrations(gs models.IntegrationService, r *mux.Router) *Integrations {
	return &Integrations{
		New:        views.NewView("bootstrap", "integrations/new"),
		EditView:   views.NewView("bootstrap", "integrations/edit"),
		IndexView:  views.NewView("bootstrap", "integrations/index"),
		ResultView: views.NewView("bootstrap", "integrations/result"),
		gs:         gs,
		r:          r,
	}
}

// Integrations contains the user's
// integrations for CRUD
type Integrations struct {
	New        *views.View
	ShowView   *views.View
	EditView   *views.View
	IndexView  *views.View
	ResultView *views.View
	gs         models.IntegrationService
	r          *mux.Router
}

// IntegrationForm is for processing the IntegrationForm
type IntegrationForm struct {
	Org   string `schema:"org"`
	Token string `schema:"token"`
}

// Index <=> GET /integrations
func (g *Integrations) Index(w http.ResponseWriter, r *http.Request) {
	user := context.User(r.Context())
	integrations, err := g.gs.ByUserID(user.ID)
	if err != nil {
		http.Error(w, "Something went wrong.", http.StatusInternalServerError)
		return
	}
	var vd views.Data
	vd.Yield = integrations
	g.IndexView.Render(w, r, vd)
}

// Create uses the POST HTTP method with path being /integrations
func (g *Integrations) Create(w http.ResponseWriter, r *http.Request) {
	var vd views.Data
	var form IntegrationForm
	if err := parseForm(r, &form); err != nil {
		vd.SetAlert(err)
		g.New.Render(w, r, vd)
		return
	}
	// This is our new code
	user := context.User(r.Context())
	integrations, err := g.gs.ByUserID(user.ID)
	if err != nil {
		http.Error(w, "Something went wrong.", http.StatusInternalServerError)
		return
	}
	if len(integrations) >= user.Limit {
		http.Redirect(w, r, "/limit", http.StatusFound)
	}
	if len(integrations) < user.Limit {
		// Then we update how we build the Integration model
		integration := models.Integration{
			Org:    form.Org,
			Token:  form.Token,
			UserID: user.ID,
		}
		if err := g.gs.Create(&integration); err != nil {
			vd.SetAlert(err)
			g.New.Render(w, r, vd)
			return
		}
		url, err := g.r.Get(EditIntegration).URL("id",
			strconv.Itoa(int(integration.ID)))
		// Check for errors creating the URL
		if err != nil {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}
		http.Redirect(w, r, url.Path, http.StatusFound)
	}

}

func unique(sSlice []string) []string {
	keys := make(map[string]bool)
	list := []string{}
	for _, entry := range sSlice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

// Scan uses the POST HTTP method with path being /integrations/scan
func (g *Integrations) Scan(w http.ResponseWriter, r *http.Request) {
	var vd views.Data
	// This is our new code
	user := context.User(r.Context())
	integrations, err := g.gs.ByUserID(user.ID)
	if err != nil {
		http.Error(w, "Something went wrong.", http.StatusInternalServerError)
		return
	}
	// Create a client connection
	ctx := C.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: integrations[0].Token},
	)
	tc := oauth2.NewClient(ctx, ts)
	client, err := github.NewEnterpriseClient("https://api.github.com", "https://api.github.com", tc)
	if err != nil {
		fmt.Printf("Problem with client %v \n", err)
	}

	// Options such as paging.
	opt := &github.RepositoryListByOrgOptions{
		ListOptions: github.ListOptions{PerPage: 10},
	}

	// To Get all pages of results...slice of repository structs
	var allRepos []*github.Repository
	var report []*Repo

	for {
		// Takes care of param 2: repo name
		repos, resp, err := client.Repositories.ListByOrg(ctx, integrations[0].Org, opt)
		// If hitting rate limit.
		if _, ok := err.(*github.RateLimitError); ok {
			log.Println("Hit rate Limit.")
		}
		// Adding the repository structs to the slice.
		allRepos = append(allRepos, repos...)
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	// (1) check if branch protections
	var unprotectedRepos []string
	for _, r := range allRepos {
		// Err in this case will mean that the branch is unprotected or it doesn't exist .. so if the branch in unprotected then protect it...? <not yet>
		prtcns, _, err := client.Repositories.GetBranchProtection(ctx, integrations[0].Org, *r.Name, *r.DefaultBranch)
		if err != nil {
			fmt.Printf("ERR: %+v", err)
		}
		// lowercase the strings
		// Use contains to look for repos of interest for repos that our PaaS is reponsible for
		// Change this to regexp
		prReview := prtcns.GetRequiredPullRequestReviews()
		reqSt := prtcns.GetRequiredStatusChecks()
		// Case 1: They do not have Pull request review enabled at all.
		if prReview == nil {
			unprotectedRepos = append(unprotectedRepos, *r.Name)
		}
		// Case 2: They do not have Required status checks enabled at all.
		if reqSt == nil {
			unprotectedRepos = append(unprotectedRepos, *r.Name)
		}
		// Case 3: They do have pull request review enabled but it is not at least 1.
		if prReview != nil && prReview.RequiredApprovingReviewCount < 1 {
			unprotectedRepos = append(unprotectedRepos, *r.Name)

		}
		// Case 4: They have Required status check but it's not set to True
		if reqSt != nil && reqSt.Strict != true {
			unprotectedRepos = append(unprotectedRepos, *r.Name)
		}
		// They are repos managed by other teams within SWAPPS like TA.
	}

	upRepos := unique(unprotectedRepos)
	for _, r := range upRepos {
		rep := &Repo{
			Name:        r,
			Protections: false,
			CQL:         false,
			Secrets:     false,
			Dependabot:  false,
		}
		report = append(report, rep)
	}
	// (2) check if dependabot enabled
	for _, r := range report {
		res, _, err := client.Dependabot.ListRepoAlerts(ctx, integrations[0].Org, r.Name, nil)
		if strings.Contains(fmt.Sprintf("%s", err), "403") {
			r.Dependabot = false
		}
		if len(res) > 0 {
			r.Dependabot = true
		}
	}

	// (3) Check if CodeQL is enabled
	for _, r := range report {
		t, _, err := client.CodeScanning.ListAlertsForRepo(ctx, integrations[0].Org, r.Name, nil)
		if strings.Contains(fmt.Sprintf("%s", err), "404") {
			r.CQL = false
		}
		if t != nil {
			r.CQL = true
		}
	}

	// (4) check if secret scanning is enabled
	for _, r := range report {
		t, _, err := client.SecretScanning.ListAlertsForRepo(ctx, integrations[0].Org, r.Name, nil)
		if strings.Contains(fmt.Sprintf("%s", err), "404") {
			r.Secrets = false
		}
		if t != nil {
			r.Secrets = true
		}
	}

	vd.Yield = report
	g.ResultView.Render(w, r, vd)

}

// Result renders result view.
func (re *Integrations) Result(w http.ResponseWriter, r *http.Request) {
	re.ResultView.Render(w, r, nil)
}

// integrationByID fetches a view of an integration by its id
func (g *Integrations) integrationByID(w http.ResponseWriter, r *http.Request) (*models.Integration, error) {
	vars := mux.Vars(r)
	idStr := vars["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid integration ID", http.StatusNotFound)
		return nil, err
	}
	integration, err := g.gs.ByID(uint(id))
	if err != nil {
		switch err {
		case models.ErrNotFound:
			http.Error(w, "Integration not found", http.StatusNotFound)
		default:
			http.Error(w, "Something went wrong.", http.StatusInternalServerError)
		}
		return nil, err
	}
	return integration, nil
}

// Edit <=> GET /integrations/:id/edit
func (g *Integrations) Edit(w http.ResponseWriter, r *http.Request) {
	integration, err := g.integrationByID(w, r)
	fmt.Printf("Edit integration: %+v", integration)
	if err != nil {
		// The integrationByID method will already render the error
		// for us, so we just need to return here.
		return
	}
	// A user needs logged in to access this page, so we can
	// assume that the RequireUser middleware has run and
	// set the user for us in the request context.
	user := context.User(r.Context())
	if integration.UserID != user.ID {
		http.Error(w, "You do not have permission to edit "+
			"this integration", http.StatusForbidden)
		return
	}
	var vd views.Data
	vd.Yield = integration
	g.EditView.Render(w, r, vd)
}

// Update an existing integration
func (g *Integrations) Update(w http.ResponseWriter, r *http.Request) {
	integration, err := g.integrationByID(w, r)
	if err != nil {
		return
	}

	user := context.User(r.Context())
	if integration.UserID != user.ID {
		http.Error(w, "Integration not found", http.StatusNotFound)
		return
	}

	var vd views.Data
	vd.Yield = integration
	var form IntegrationForm
	if err := parseForm(r, &form); err != nil {
		// If there is an error we are going to render the
		// EditView again with an alert message.
		vd.SetAlert(err)
		g.EditView.Render(w, r, vd)
		return
	}
	integration.Org = form.Org
	err = g.gs.Update(integration)
	// If there is an error our alert will be an error. Otherwise
	// we will still render an alert, but instead it will be
	// a success message.
	if err != nil {
		vd.SetAlert(err)
	} else {
		vd.Alert = &views.Alert{
			Level:   views.AlertLvlSuccess,
			Message: "Integration successfully updated!",
		}
	}
	// Error or not, we are going to render the EditView with
	// our updated information.
	http.Redirect(w, r, "/integrations", http.StatusFound)
}

// Delete <=> POST /integrations/:id/delete
func (g *Integrations) Delete(w http.ResponseWriter, r *http.Request) {
	// Lookup the integration using the integrationByID we wrote earlier
	integration, err := g.integrationByID(w, r)
	if err != nil {
		// If there is an error the integrationByID will have rendered
		// it for us already.
		return
	}
	// We also need to retrieve the user and verify they have
	// permission to delete this integration. This means we will
	// need to use the RequireUser middleware on any routes
	// mapped to this method.
	user := context.User(r.Context())
	if integration.UserID != user.ID {
		http.Error(w, "You do not have permission to edit "+
			"this integration", http.StatusForbidden)
		return
	}

	var vd views.Data
	err = g.gs.Delete(integration.ID)
	if err != nil {
		// If there is an error we want to set an alert and
		// render the edit page with the error. We also need
		// to set the Yield to integration so that the EditView
		// is rendered correctly.
		vd.SetAlert(err)
		vd.Yield = integration
		g.EditView.Render(w, r, vd)
		return
	}
	// TODO: We will eventually want to redirect to the index
	// page that lists all integrations this user owns, but for
	// now a success message will suffice.
	url, err := g.r.Get(IndexIntegrations).URL()
	if err != nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	http.Redirect(w, r, url.Path, http.StatusFound)
}
