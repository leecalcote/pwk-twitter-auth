package handlers

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/leecalcote/pwk-twitter-auth/dao"
	"github.com/leecalcote/pwk-twitter-auth/queue"
	"github.com/leecalcote/pwk-twitter-auth/tweeter"

	oauth1Login "github.com/dghubble/gologin/oauth1"
	"github.com/dghubble/gologin/twitter"
	"github.com/dghubble/oauth1"
	twitterOAuth1 "github.com/dghubble/oauth1/twitter"
	"github.com/dghubble/sessions"
	"github.com/sirupsen/logrus"
)

const (
	sessionUserKey       = "twitterID"
	sessionTwitterToken  = "token"
	sessionTwitterSecret = "secret"
	cookieSuffix         = "_referrer"
)

// sessionStore encodes and decodes session data stored in signed cookies
var (
	sessionName   = os.Getenv("EVENT")
	sessionSecret = base64.StdEncoding.EncodeToString([]byte(sessionName))
	sessionStore  = sessions.NewCookieStore([]byte(sessionSecret), nil)
	cookieName    = os.Getenv("EVENT") + cookieSuffix
)

// Config configures the main ServeMux.
type Config struct {
	TwitterConsumerKey    string
	TwitterConsumerSecret string
	CallbackURL           string

	Mmq *queue.MemQ

	Loc *time.Location
}

// New returns a new ServeMux with app routes.
func New(config *Config) *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("/twitter/profile", authMiddleware(http.HandlerFunc(profileHandler)))
	mux.Handle("/twitter/tweet", authMiddleware(http.HandlerFunc(tweetHandler(config))))
	mux.HandleFunc("/twitter/logout", logoutHandler)

	oauth1Config := &oauth1.Config{
		ConsumerKey:    config.TwitterConsumerKey,
		ConsumerSecret: config.TwitterConsumerSecret,
		CallbackURL:    config.CallbackURL,
		Endpoint:       twitterOAuth1.AuthorizeEndpoint,
	}
	mux.Handle("/twitter/login", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		source := r.URL.Query().Get("source")
		logrus.Infof("initiating twitter login with source: %s", source)

		src, err := base64.URLEncoding.DecodeString(source)
		if err != nil {
			logrus.Errorf("Base64 URL Decode Error: %v", err)
			http.Error(w, "source provided is not valid", http.StatusForbidden)
			return
		}
		tu, err := url.Parse(string(src))
		if err != nil {
			logrus.Errorf("Source URL Parse Error: %v", err)
			http.Error(w, "source provided is not valid", http.StatusForbidden)
			return
		}
		// if tu.Path == "/productpage" {
		// tu.Path = "/login"
		ck := &http.Cookie{
			Name:    cookieName,
			Value:   tu.String(),
			Expires: time.Now().Add(5 * time.Minute),
			Path:    "/",
		}
		http.SetCookie(w, ck)
		logrus.Infof("set cookie with value: %s", tu.String())
		// }
		// }
		twitter.LoginHandler(oauth1Config, nil).ServeHTTP(w, r)
	}))
	mux.Handle("/twitter/callback", twitter.CallbackHandler(oauth1Config, issueSession(config), nil))

	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	}))

	return mux
}

// issueSession issues a cookie session after successful Twitter login
func issueSession(config *Config) http.Handler {
	fn := func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		twitterUser, err := twitter.UserFromContext(ctx)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		logrus.Infof("Twitter user: %+#v", twitterUser)
		accessToken, accessSecret, err := oauth1Login.AccessTokenFromContext(ctx)
		logrus.Infof("Twitter Access token: %s, Secret: %s", accessToken, accessSecret)

		session := sessionStore.New(sessionName)
		session.Values[sessionUserKey] = twitterUser.ID
		session.Values[sessionTwitterToken] = accessToken
		session.Values[sessionTwitterSecret] = accessSecret
		session.Save(w)

		postTweet(config, req, fmt.Sprintf(`I've run my first @IstioMesh destination rule with user-based routing! Thank you, #%s ! 

		//@layer5
		`, os.Getenv("EVENT")))

		ck, err := req.Cookie(cookieName)
		if err != nil {
			logrus.Errorf("error: unable to find the referrer cookie: %v", err)
			http.Redirect(w, req, "/twitter/profile", http.StatusFound)
			return
		}

		ck.Expires = time.Now().Add(-2 * time.Second)
		http.SetCookie(w, ck)
		cku, _ := url.Parse(ck.Value)
		qp := cku.Query()
		qp.Add("username", twitterUser.Name)
		cke := w.Header().Get("Set-Cookie")
		ckePieces := strings.Split(cke, "; ")
		ckValue := ""
		for _, ckep := range ckePieces {
			if strings.HasPrefix(ckep, sessionName) {
				ckValue = strings.TrimPrefix(ckep, sessionName+"=")
			}
		}
		logrus.Infof("Attempting to get cookie before response is sent out: %s", ckValue)
		qp.Add(sessionName, ckValue)
		cku.RawQuery = qp.Encode()
		http.Redirect(w, req, cku.String(), http.StatusFound)
	}
	return http.HandlerFunc(fn)
}

// profileHandler shows protected user content.
func profileHandler(w http.ResponseWriter, req *http.Request) {
	fmt.Fprint(w, `<p>You are logged in!</p><form action="/twitter/logout" method="post"><input type="submit" value="Logout"></form>`)
}

func tweetHandler(config *Config) func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		err := req.ParseForm()
		if err != nil {
			http.Error(w, "unable to process the received data", http.StatusForbidden)
			return
		}
		logrus.Infof("Received form: %v", req.Form)
		postTweet(config, req, req.FormValue("msg"))
	}
}

// logoutHandler destroys the session on POSTs and redirects to home.
func logoutHandler(w http.ResponseWriter, req *http.Request) {
	if req.Method == "POST" {
		sessionStore.Destroy(w, sessionName)
	}
	http.Redirect(w, req, req.Referer(), http.StatusFound)
}

func authMiddleware(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, req *http.Request) {
		if !validateAuth(req) {
			http.Redirect(w, req, "/twitter/login", http.StatusFound)
			return
		}
		next.ServeHTTP(w, req)
	}
	return http.HandlerFunc(fn)
}

func validateAuth(req *http.Request) bool {
	if _, err := sessionStore.Get(req, sessionName); err == nil {
		return true
	}
	return false
}

func postTweet(config *Config, req *http.Request, msg string) error {
	if strings.TrimSpace(msg) == "" {
		logrus.Error("Error: message is empty")
		return errors.New("message is empty")
	}
	ctx := req.Context()
	twitterUser, err := twitter.UserFromContext(ctx)
	if err != nil {
		return errors.New("unable to find twitter user")
	}

	session, err := sessionStore.Get(req, sessionName)
	if err != nil {
		return errors.New("unable to get session data")
	}
	accessToken, _ := session.Values[sessionTwitterToken].(string)
	accessSecret, _ := session.Values[sessionTwitterSecret].(string)

	err = config.Mmq.DepoInQueue(&dao.TwitterUser{
		Event:     sessionName,
		Email:     twitterUser.Email,
		ID:        twitterUser.ScreenName,
		Name:      twitterUser.Name,
		LoginTime: time.Now(),
	}, &tweeter.Tweeter{
		ConsumerKey:    config.TwitterConsumerKey,
		ConsumerSecret: config.TwitterConsumerSecret,

		AccessToken:  accessToken,
		AccessSecret: accessSecret,
	}, msg)
	if err != nil {
		logrus.Errorf("error posting to queue: %v", err)
		return errors.New("unable to initiate the twitter post flow")
	}
	return nil
}
