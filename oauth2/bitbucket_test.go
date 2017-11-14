// Implementation of Bitbucket Oauth2 provider tests.
//
// Because Bitbucket API does not return the email in the /user request, it's necessary
// also to request /user/emails endpoint. This means for the tests it's necessary to
// cover also possible errors regarding to both endpoint.
//
// This tests use a setupHandler method that allows to create different handlers
// based on the HTTP response needed to generate specific errors. An enum of HandleState is
// declared to define different possible states for handlers.
//

package oauth2

import (
	. "github.com/stretchr/testify/assert"
	"net/http/httptest"
	"testing"
	"github.com/gorilla/mux"
	"encoding/json"
	"net/http"
)

// bitbucketTestUserResponse response for /user endpoint
var bitbucketTestUserResponse = `{
  "created_on": "2011-12-20T16:34:07.132459+00:00",
  "display_name": "tutorials account",
  "is_staff": false,
  "links": {
    "avatar": {
      "href": "https://bitbucket.org/account/tutorials/avatar/32/"
    },
    "followers": {
      "href": "https://api.bitbucket.org/2.0/users/tutorials/followers"
    },
    "following": {
      "href": "https://api.bitbucket.org/2.0/users/tutorials/following"
    },
    "hooks": {
      "href": "https://api.bitbucket.org/2.0/users/tutorials/hooks"
    },
    "html": {
      "href": "https://bitbucket.org/tutorials/"
    },
    "repositories": {
      "href": "https://api.bitbucket.org/2.0/repositories/tutorials"
    },
    "self": {
      "href": "https://api.bitbucket.org/2.0/users/tutorials"
    },
    "snippets": {
      "href": "https://api.bitbucket.org/2.0/snippets/tutorials"
    }
  },
  "location": null,
  "type": "user",
  "username": "tutorials",
  "uuid": "{c788b2da-b7a2-404c-9e26-d3f077557007}",
  "website": "https://tutorials.bitbucket.org/"
}`

// bitbucketTestUserEmailResponse response for /user/emails endpoint
var bitbucketTestUserEmailResponse = `{
  "page": 1,
  "pagelen": 10,
  "size": 1,
  "values": [
    {
      "email": "tutorials@bitbucket.com",
      "is_confirmed": true,
      "is_primary": true,
      "links": {
        "self": {
          "href": "https://api.bitbucket.org/2.0/user/emails/tutorials@bitbucket.com"
        }
      },
      "type": "email"
    },
	{
      "email": "anotheremail@bitbucket.com",
      "is_confirmed": false,
      "is_primary": false,
      "links": {
        "self": {
          "href": "https://api.bitbucket.org/2.0/user/emails/anotheremail@bitbucket.com"
        }
      },
      "type": "email"
    }
  ]
}`

// bitbucketTestEmptyEmailResponse empty response for /user endpoint
var bitbucketTestEmptyEmailResponse = `{
  "page": 1,
  "pagelen": 10,
  "size": 0,
  "values": []
}`

// Enum to define multiple type of Handler States
type HandlerState int
const (
	Success HandlerState = iota
	WrongContentType
	StatusCodeNotOK
	NotJsonContent
	HttpError

)

// setupHandler returns a Handler based on the Handler State and the given response.
func setupHandler(handlerState HandlerState, response string) http.HandlerFunc {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Content-Type", "application/json; charset=utf-8")

		switch handlerState {
		case Success:
			w.Write([]byte(response))
		case WrongContentType:
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write([]byte(response))
		case StatusCodeNotOK:
			w.WriteHeader(http.StatusConflict)
			w.Write([]byte(response))
		case NotJsonContent:
			w.Write([]byte(nil))
		case HttpError:
			panic("error calling http method")
		}
	})
	return handler
}

// getServer Returns a server with two routes /user managed by userHandler and /user/email managed by emailhandler
func getServer(userHandler http.HandlerFunc, emailHandler http.HandlerFunc) *httptest.Server {
	r := mux.NewRouter()

	r.HandleFunc("/user", userHandler)
	r.HandleFunc("/user/emails", emailHandler)

	return httptest.NewServer(r)
}

// Test_Bitbucket_getUserInfo tests Bitbucket provider returns the expected information
func Test_Bitbucket_getUserInfo(t *testing.T) {
	server := getServer(
		setupHandler(Success, bitbucketTestUserResponse),
		setupHandler(Success, bitbucketTestUserEmailResponse),
	)
	defer server.Close()

	bitbucketAPI = server.URL

	u, rawJSON, err := providerBitbucket.GetUserInfo(TokenInfo{AccessToken: "secret"})
	NoError(t, err)
	Equal(t, "tutorials", u.Sub)
	Equal(t, "tutorials@bitbucket.com", u.Email)
	Equal(t, "tutorials account", u.Name)
	Equal(t, bitbucketTestUserResponse, rawJSON)
}

// Test_Bitbucket_wrongContentTypeOnUser tests if the provider fails in the proper way when the /user endpoint returns a bad content-type
func Test_Bitbucket_wrongContentTypeOnUser(t *testing.T){
	server := getServer(
		setupHandler(WrongContentType, bitbucketTestUserResponse),
		setupHandler(Success, bitbucketTestUserEmailResponse),
	)
	defer server.Close()

	bitbucketAPI = server.URL

	u, _, err := providerBitbucket.GetUserInfo(TokenInfo{AccessToken: "secret"})
	Error(t, err)
	Empty(t, u.Email)
}

// Test_Bitbucket_httpStatusNotOKOnUser tests if the provider fails in the proper way when the /user endpoint returns a non OK status
func Test_Bitbucket_httpStatusNotOKOnUser(t *testing.T){
	server := getServer(
		setupHandler(StatusCodeNotOK, bitbucketTestUserResponse),
		setupHandler(Success, bitbucketTestUserEmailResponse),
	)
	defer server.Close()

	bitbucketAPI = server.URL

	u, _, err := providerBitbucket.GetUserInfo(TokenInfo{AccessToken: "secret"})
	Error(t, err)
	Empty(t, u.Email)
}

// Test_Bitbucket_noJsonContentOnUser tests if the provider fails in the proper way when the /user endpoint returns a non Json Content
func Test_Bitbucket_noJsonContentOnUser(t *testing.T){
	server := getServer(
		setupHandler(NotJsonContent, bitbucketTestUserResponse),
		setupHandler(Success, bitbucketTestUserEmailResponse),
	)
	defer server.Close()

	bitbucketAPI = server.URL

	u, _, err := providerBitbucket.GetUserInfo(TokenInfo{AccessToken: "secret"})
	Error(t, err)
	Empty(t, u.Email)
}

// Test_Bitbucket_httpErrorOnUser tests if the provider fails in the proper way when is not possible to call the /user endpoint
func Test_Bitbucket_httpErrorOnUser(t *testing.T){
	server := getServer(
		setupHandler(HttpError, bitbucketTestUserResponse),
		setupHandler(Success, bitbucketTestUserEmailResponse),
	)
	defer server.Close()

	bitbucketAPI = server.URL

	u, _, err := providerBitbucket.GetUserInfo(TokenInfo{AccessToken: "secret"})
	Error(t, err)
	Empty(t, u.Email)
}

// Test_Bitbucket_wrongContentTypeOnEmail tests if the provider fails in the proper way when the /user/emails endpoint returns a bad content-type
func Test_Bitbucket_wrongContentTypeOnEmail(t *testing.T){
	server := getServer(
		setupHandler(Success, bitbucketTestUserResponse),
		setupHandler(WrongContentType, bitbucketTestUserEmailResponse),
	)
	defer server.Close()

	bitbucketAPI = server.URL

	u, _, err := providerBitbucket.GetUserInfo(TokenInfo{AccessToken: "secret"})
	Error(t, err)
	Empty(t, u.Email)
}

// Test_Bitbucket_httpStatusNotOKOnEmail tests if the provider fails in the proper way when the /user/emails endpoint returns a non OK status
func Test_Bitbucket_httpStatusNotOKOnEmail(t *testing.T){
	server := getServer(
		setupHandler(Success, bitbucketTestUserResponse),
		setupHandler(StatusCodeNotOK, bitbucketTestUserEmailResponse),
	)
	defer server.Close()

	bitbucketAPI = server.URL

	u, _, err := providerBitbucket.GetUserInfo(TokenInfo{AccessToken: "secret"})
	Error(t, err)
	Empty(t, u.Email)
}

// Test_Bitbucket_noJsonContentOnEmail tests if the provider fails in the proper way when the /user/emails endpoint returns a non Json Content
func Test_Bitbucket_noJsonContentOnEmail(t *testing.T){
	server := getServer(
		setupHandler(Success, bitbucketTestUserResponse),
		setupHandler(NotJsonContent, bitbucketTestUserEmailResponse),
	)
	defer server.Close()

	bitbucketAPI = server.URL

	u, _, err := providerBitbucket.GetUserInfo(TokenInfo{AccessToken: "secret"})
	Error(t, err)
	Empty(t, u.Email)
}

// Test_Bitbucket_httpErrorEmail tests if the provider fails in the proper way when is not possible to call the /user/emails endpoint
func Test_Bitbucket_httpErrorEmail(t *testing.T){
	server := getServer(
		setupHandler(Success, bitbucketTestUserResponse),
		setupHandler(HttpError, bitbucketTestUserEmailResponse),
	)
	defer server.Close()

	bitbucketAPI = server.URL

	u, _, err := providerBitbucket.GetUserInfo(TokenInfo{AccessToken: "secret"})
	Error(t, err)
	Empty(t, u.Email)
}

// Test_Bitbucket_emptyEmailResponse tests if the provider returns the correct answer when there's an empty list of mails returned by /user/emails
func Test_Bitbucket_emptyEmailResponse(t *testing.T) {
	server := getServer(
		setupHandler(Success, bitbucketTestUserResponse),
		setupHandler(Success, bitbucketTestEmptyEmailResponse),
	)
	defer server.Close()

	bitbucketAPI = server.URL

	u, _, err := providerBitbucket.GetUserInfo(TokenInfo{AccessToken: "secret"})
	NoError(t, err)
	Equal(t, "tutorials", u.Sub)
	Equal(t, "", u.Email)
	Equal(t, "tutorials account", u.Name)
}

// Test_Bitbucket_getPrimaryEmailAddress tests the returned primary email is the expected email
func Test_Bitbucket_getPrimaryEmailAddress(t *testing.T)  {
	userEmails := emails{}
	err := json.Unmarshal([]byte(bitbucketTestUserEmailResponse), &userEmails)
	NoError(t, err)
	Equal(t,"tutorials@bitbucket.com", userEmails.getPrimaryEmailAddress())
}
