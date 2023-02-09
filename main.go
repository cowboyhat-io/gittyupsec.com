package main

import "C"
import (
	"flag"
	"fmt"
	"github.com/cowboyhat-io/gittyupsec.com/rand"
	"github.com/gorilla/csrf"
	"net/http"

	"github.com/cowboyhat-io/gittyupsec.com/controllers"
	"github.com/cowboyhat-io/gittyupsec.com/middleware"
	models "github.com/cowboyhat-io/gittyupsec.com/models"
	"github.com/cowboyhat-io/gittyupsec.com/views"

	"github.com/gorilla/mux"
)

var (
	pricingView *views.View
	homeView    *views.View
)

func notFound(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, "<h1>404</h1><p>A tiny error has occurred.</p>")
}

func main() {
	boolPtr := flag.Bool("prod", false, "Provide this flag "+
		"in production. This ensures that a .config file is "+
		"provided before the application starts.")
	flag.Parse()
	// boolPtr is a pointer to a boolean, so we need to use
	// *boolPtr to get the boolean value and pass it into our
	// LoadConfig function
	cfg := LoadConfig(*boolPtr)
	dbCfg := DefaultPostgresConfig()
	// This isn't complete, but we will come back to it shortly
	services, err := models.NewServices(
		models.WithGorm(dbCfg.Dialect(), dbCfg.ConnectionInfo()),
		// Only log when not in prod
		models.WithLogMode(!cfg.IsProd()),
		models.WithUser(cfg.Pepper, cfg.HMACKey),
		models.WithIntegration(),
	)
	if err != nil {
		panic(err)
	}
	defer services.Close()
	services.AutoMigrate()
	// services.DestructiveReset()
	r := mux.NewRouter()
	staticC := controllers.NewStatic()
	usersC := controllers.NewUsers(services.User)
	integrationsC := controllers.NewIntegrations(services.Integration, r)
	userMw := middleware.User{
		UserService: services.User,
	}

	requireUserMw := middleware.RequireUser{}

	// integration
	newIntegration := requireUserMw.Apply(integrationsC.New)
	createIntegration := requireUserMw.ApplyFn(integrationsC.Create)

	// Creates an implementation of the http.Handler interface
	var four http.Handler = http.HandlerFunc(notFound)
	// assign it to the not found handler
	r.NotFoundHandler = four
	r.Handle("/", staticC.Home).Methods("GET")
	r.Handle("/limit", staticC.Limit).Methods("GET")
	r.Handle("/automatic", staticC.Automatic).Methods("GET")
	r.Handle("/business", staticC.Business).Methods("GET")
	r.Handle("/github-data-breaches", staticC.Breaches).Methods("GET")
	r.Handle("/privacy", staticC.Privacy).Methods("GET")
	r.Handle("/team", staticC.Team).Methods("GET")

	// Integrations (save a github token...)
	r.Handle("/integrations/new", newIntegration).Methods("GET")
	r.HandleFunc("/integrations", createIntegration).Methods("POST").Name(controllers.IndexIntegrations)
	r.HandleFunc("/integrations/{id:[0-9]+}/edit", requireUserMw.ApplyFn(integrationsC.Edit)).Methods("GET").Name(controllers.EditIntegration)
	r.HandleFunc("/integrations/{id:[0-9]+}/update", requireUserMw.ApplyFn(integrationsC.Update)).Methods("POST")
	r.HandleFunc("/integrations/{id:[0-9]+}/delete", requireUserMw.ApplyFn(integrationsC.Delete)).Methods("POST")
	r.HandleFunc("/integrations", requireUserMw.ApplyFn(integrationsC.Index)).Methods("GET")
	// integrations/scan -> POST -> imports repos & scans them
	r.HandleFunc("/integrations/scan", requireUserMw.ApplyFn(integrationsC.Result)).Methods("GET")
	r.HandleFunc("/integrations/scan", requireUserMw.ApplyFn(integrationsC.Scan)).Methods("POST")

	// Users
	r.Handle("/pricing", staticC.Pricing).Methods("GET")
	r.HandleFunc("/signup", usersC.New).Methods("GET")
	r.HandleFunc("/signup", usersC.Create).Methods("POST")
	r.Handle("/login", usersC.LoginView).Methods("GET")
	r.Handle("/logout", requireUserMw.ApplyFn(usersC.Logout)).Methods("POST")
	r.HandleFunc("/login", usersC.Login).Methods("POST")
	r.HandleFunc("/cookietest", usersC.CookieTest).Methods("GET")

	b, err := rand.Bytes(32)
	if err != nil {
		panic(err)
	}
	staticDir := "/public/"
	csrfMw := csrf.Protect(b, csrf.Secure(cfg.IsProd()))
	r.PathPrefix(staticDir).Handler(http.StripPrefix(staticDir, http.FileServer(http.Dir("."+staticDir))))
	http.ListenAndServe(fmt.Sprintf(":%d", cfg.Port), csrfMw(userMw.Apply(r)))
}
