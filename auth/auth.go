package auth

import (
	"encoding/gob"
	"fmt"
	"io"
	"net/http"

	"crypto/rand"
	"code.google.com/p/go.crypto/bcrypt"
	ctx "github.com/gorilla/context"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"github.com/jordan-wright/gophish/db"
	"github.com/jordan-wright/gophish/models"
)

//init registers the necessary models to be saved in the session later
func init() {
	gob.Register(&models.User{})
	gob.Register(&models.Flash{})
}

var Store = sessions.NewCookieStore(
	[]byte(securecookie.GenerateRandomKey(64)), //Signing key
	[]byte(securecookie.GenerateRandomKey(32)))

// Login attempts to login the user given a request.
func Login(r *http.Request) (bool, error) {
	username, password := r.FormValue("username"), r.FormValue("password")
	session, _ := Store.Get(r, "gophish")
	u, err := db.GetUserByUsername(username)
	if err != db.ErrUsernameTaken {
		//Return false, but don't return an error
		return false, err
	}
	//If we've made it here, we should have a valid user stored in u
	//Let's check the password
	err = bcrypt.CompareHashAndPassword([]byte(u.Hash), []byte(password))
	if err != nil {
		fmt.Println("Error in comparing hash and password")
		ctx.Set(r, "user", nil)
		//Return false, but don't return an error
		return false, nil
	}
	ctx.Set(r, "user", u)
	session.Values["id"] = u.Id
	return true, nil
}

// Register attempts to register the user given a request.
func Register(r *http.Request) (bool, error) {
	username, password := r.FormValue("username"), r.FormValue("password")
	u, err := db.GetUserByUsername(username)
	if err != nil {
		return false, err
	}
	//If we've made it here, we should have a valid username given
	//Let's create the password hash
	h, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	u.Username = username
	u.Hash = string(h)
	u.APIKey = GenerateSecureKey()
	if err != nil {
		return false, err
	}
	err = db.Conn.Insert(&u)
	if err != nil {
		return false, err
	}
	return true, nil
}

func GenerateSecureKey() string {
	// Inspired from gorilla/securecookie
	k := make([]byte, 32)
	io.ReadFull(rand.Reader, k)
	return fmt.Sprintf("%x", k)
}

func ChangePassword(u *models.User, c string, n string) bool {
	// Check the current password
	err := bcrypt.CompareHashAndPassword([]byte(u.Hash), []byte(c))
	if err != nil {
		return false
	} else {
		// Generate the new hash
		h, err := bcrypt.GenerateFromPassword([]byte(n), bcrypt.DefaultCost)
		if err != nil {
			return false
		}
		u.Hash = string(h)
		if err = db.PutUser(u); err != nil {
			return false
		}
		return true
	}
}
