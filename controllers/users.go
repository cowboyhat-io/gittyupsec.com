package controllers

import (
	"fmt"
	"github.com/cowboyhat-io/gittyupsec.com/context"
	"net/http"
	"time"

	"github.com/cowboyhat-io/gittyupsec.com/models"
	"github.com/cowboyhat-io/gittyupsec.com/rand"
	"github.com/cowboyhat-io/gittyupsec.com/views"
)

const (
	LIMIT = 1
)

// Users presents the view
// and what a user (us) can do
type Users struct {
	NewView   *views.View
	LoginView *views.View
	us        models.UserService
}

// SignupForm holds the expected data
// for the sign-up form for users
// to interact with
type SignupForm struct {
	UserName string `schema:"username"`
	Email    string `schema:"email"`
	Password string `schema:"password"`
}

// LoginForm holds the expected data
// for the login in form for returning
// users
type LoginForm struct {
	Email    string `schema:"email"`
	Password string `schema:"password"`
}

// Login processes the user's login attempt to the
// application
func (u *Users) Login(w http.ResponseWriter, r *http.Request) {
	var vd views.Data
	var form LoginForm
	if err := parseForm(r, &form); err != nil {
		vd.SetAlert(err)
		u.LoginView.Render(w, r, vd)
		return
	}
	user, err := u.us.Authenticate(form.Email, form.Password)
	if err != nil {
		switch err {
		case models.ErrNotFound:
			vd.AlertError("No user exists with that email address")
		default:
			vd.SetAlert(err)
		}
		u.LoginView.Render(w, r, vd)
		return
	}
	// Add this code at the end of the method
	err = u.signIn(w, user)
	if err != nil {
		vd.SetAlert(err)
		u.LoginView.Render(w, r, vd)
		return
	}
	http.Redirect(w, r, "/integrations", http.StatusFound)
}

func (u *Users) signIn(w http.ResponseWriter, user *models.User) error {
	if user.Remember == "" {
		token, err := rand.RememberToken()
		if err != nil {
			return err
		}
		user.Remember = token
		err = u.us.Update(user)
		if err != nil {
			return err
		}
	}

	cookie := http.Cookie{
		Name:     "remember_token",
		Value:    user.Remember,
		HttpOnly: true,
	}
	http.SetCookie(w, &cookie)
	return nil
}

// CookieTest is for debugging in dev. to see that
// the user data is being stored and returned
func (u *Users) CookieTest(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("remember_token")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	user, err := u.us.ByRemember(cookie.Value)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Fprintln(w, user)
}

// NewUsers defines the views
// for users that do not have an account
func NewUsers(us models.UserService) *Users {
	return &Users{
		NewView:   views.NewView("bootstrap", "users/new"),
		LoginView: views.NewView("bootstrap", "users/login"),
		us:        us,
	}
}

// New Renders the NewView for users
func (u *Users) New(w http.ResponseWriter, r *http.Request) {
	u.NewView.Render(w, r, nil)
}

// Create processes the sign-up form and stores a
// new user record in the database
func (u *Users) Create(w http.ResponseWriter, r *http.Request) {
	var vd views.Data
	var form SignupForm
	if err := parseForm(r, &form); err != nil {
		vd.SetAlert(err)
		u.NewView.Render(w, r, vd)
		return
	}
	user := models.User{
		UserName: form.UserName,
		Email:    form.Email,
		Password: form.Password,
		Limit:    LIMIT,
	}
	if err := u.us.Create(&user); err != nil {
		vd.SetAlert(err)
		u.NewView.Render(w, r, vd)
		return
	}
	err := u.signIn(w, &user)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}
	http.Redirect(w, r, "/integrations", http.StatusFound)
}

// Logout is used to delete a user's session cookie
// and invalidate their current remember token, which will
// sign the current user out.
//
// POST /logout
func (u *Users) Logout(w http.ResponseWriter, r *http.Request) {
	// First expire the user's cookie
	cookie := http.Cookie{
		Name:     "remember_token",
		Value:    "",
		Expires:  time.Now(),
		HttpOnly: true,
	}
	http.SetCookie(w, &cookie)
	// Then we update the user with a new remember token
	user := context.User(r.Context())
	// We are ignoring errors for now because they are
	// unlikely, and even if they do occur we can't recover
	// now that the user doesn't have a valid cookie
	token, _ := rand.RememberToken()
	user.Remember = token
	u.us.Update(user)
	// Finally send the user to the home page
	http.Redirect(w, r, "/", http.StatusFound)
}
