package ui

import (
	"encoding/json"
	"html/template"
	"net/http"
	"path/filepath"
	"regexp"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/supergiant/supergiant/pkg/client"
	"github.com/supergiant/supergiant/pkg/core"
	"github.com/supergiant/supergiant/pkg/model"
)

var templates = make(map[string]*template.Template)

func init() {
	partials, err := filepath.Glob("ui/views/partials/*.html")
	if err != nil {
		panic(err)
	}
	layouts, err := filepath.Glob("ui/views/*.html")
	if err != nil {
		panic(err)
	}
	for _, partial := range partials {
		key := regexp.MustCompile(`([^/]+)\.html$`).FindStringSubmatch(partial)[1]
		templates[key] = template.Must(template.ParseFiles(append(layouts, partial)...))
	}
}

func NewRouter(c *core.Core, baseRouter *mux.Router) *mux.Router {
	base := baseRouter.StrictSlash(true)

	base.HandleFunc("/", uiRedirect).Methods("GET")

	r := base.PathPrefix("/ui").Subrouter()

	r.HandleFunc("/", restrictedHandler(c, Root)).Methods("GET")

	// Login-based functions can't be authenticated
	sharedClient := c.NewAPIClient("", "")
	r.HandleFunc("/sessions/new", openHandler(sharedClient, NewSession)).Methods("GET")
	r.HandleFunc("/sessions", openHandler(sharedClient, CreateSession)).Methods("POST")
	r.HandleFunc("/sessions/{id}", openHandler(sharedClient, GetSession)).Methods("GET")

	r.HandleFunc("/sessions", restrictedHandler(c, ListSessions)).Methods("GET")
	r.HandleFunc("/sessions/{id}/delete", restrictedHandler(c, DeleteSession)).Methods("PUT")

	r.HandleFunc("/users/new", restrictedHandler(c, NewUser)).Methods("GET")
	r.HandleFunc("/users", restrictedHandler(c, CreateUser)).Methods("POST")
	r.HandleFunc("/users", restrictedHandler(c, ListUsers)).Methods("GET")
	r.HandleFunc("/users/{id}", restrictedHandler(c, GetUser)).Methods("GET")
	r.HandleFunc("/users/{id}/edit", restrictedHandler(c, EditUser)).Methods("GET")
	r.HandleFunc("/users/{id}", restrictedHandler(c, UpdateUser)).Methods("POST")
	r.HandleFunc("/users/{id}/delete", restrictedHandler(c, DeleteUser)).Methods("PUT")
	r.HandleFunc("/users/{id}/regenerate_api_token", restrictedHandler(c, RegenerateUserAPIToken)).Methods("PUT")

	r.HandleFunc("/cloud_accounts/new", restrictedHandler(c, NewCloudAccount)).Methods("GET")
	r.HandleFunc("/cloud_accounts", restrictedHandler(c, CreateCloudAccount)).Methods("POST")
	r.HandleFunc("/cloud_accounts", restrictedHandler(c, ListCloudAccounts)).Methods("GET")
	r.HandleFunc("/cloud_accounts/{id}", restrictedHandler(c, GetCloudAccount)).Methods("GET")
	r.HandleFunc("/cloud_accounts/{id}/delete", restrictedHandler(c, DeleteCloudAccount)).Methods("PUT")

	r.HandleFunc("/kubes/new", restrictedHandler(c, NewKube)).Methods("GET")
	r.HandleFunc("/kubes", restrictedHandler(c, CreateKube)).Methods("POST")
	r.HandleFunc("/kubes", restrictedHandler(c, ListKubes)).Methods("GET")
	r.HandleFunc("/kubes/{id}", restrictedHandler(c, GetKube)).Methods("GET")
	r.HandleFunc("/kubes/{id}/delete", restrictedHandler(c, DeleteKube)).Methods("PUT")

	r.HandleFunc("/nodes/new", restrictedHandler(c, NewNode)).Methods("GET")
	r.HandleFunc("/nodes", restrictedHandler(c, CreateNode)).Methods("POST")
	r.HandleFunc("/nodes", restrictedHandler(c, ListNodes)).Methods("GET")
	r.HandleFunc("/nodes/{id}", restrictedHandler(c, GetNode)).Methods("GET")
	r.HandleFunc("/nodes/{id}/delete", restrictedHandler(c, DeleteNode)).Methods("PUT")

	r.HandleFunc("/volumes/new", restrictedHandler(c, NewVolume)).Methods("GET")
	r.HandleFunc("/volumes", restrictedHandler(c, CreateVolume)).Methods("POST")
	r.HandleFunc("/volumes", restrictedHandler(c, ListVolumes)).Methods("GET")
	r.HandleFunc("/volumes/{id}", restrictedHandler(c, GetVolume)).Methods("GET")
	r.HandleFunc("/volumes/{id}/edit", restrictedHandler(c, EditVolume)).Methods("GET")
	r.HandleFunc("/volumes/{id}", restrictedHandler(c, UpdateVolume)).Methods("POST")
	r.HandleFunc("/volumes/{id}/delete", restrictedHandler(c, DeleteVolume)).Methods("PUT")

	r.HandleFunc("/entrypoints/new", restrictedHandler(c, NewEntrypoint)).Methods("GET")
	r.HandleFunc("/entrypoints", restrictedHandler(c, CreateEntrypoint)).Methods("POST")
	r.HandleFunc("/entrypoints", restrictedHandler(c, ListEntrypoints)).Methods("GET")
	r.HandleFunc("/entrypoints/{id}", restrictedHandler(c, GetEntrypoint)).Methods("GET")
	r.HandleFunc("/entrypoints/{id}/delete", restrictedHandler(c, DeleteEntrypoint)).Methods("PUT")

	r.HandleFunc("/entrypoint_listeners/new", restrictedHandler(c, NewEntrypointListener)).Methods("GET")
	r.HandleFunc("/entrypoint_listeners", restrictedHandler(c, CreateEntrypointListener)).Methods("POST")
	r.HandleFunc("/entrypoint_listeners", restrictedHandler(c, ListEntrypointListeners)).Methods("GET")
	r.HandleFunc("/entrypoint_listeners/{id}", restrictedHandler(c, GetEntrypointListener)).Methods("GET")
	r.HandleFunc("/entrypoint_listeners/{id}/delete", restrictedHandler(c, DeleteEntrypointListener)).Methods("PUT")

	return baseRouter
}

func restrictedHandler(c *core.Core, fn func(*client.Client, http.ResponseWriter, *http.Request) error) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// Load Client by Session ID stored in cookie. If either cookie or client
		// does not exist, redirect to login page with 401.
		var sessionID string
		var client *client.Client
		if sessionCookie, err := r.Cookie(core.SessionCookieName); err == nil {
			sessionID = sessionCookie.Value
			client = c.Sessions.Client(sessionID)
		}
		if client == nil {
			http.Redirect(w, r, "/ui/sessions/new", http.StatusFound) // can't do 401 here unless you want browser behavior
			return
		}
		if err := fn(client, w, r); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
		}
	}
}

func openHandler(sharedClient *client.Client, fn func(*client.Client, http.ResponseWriter, *http.Request) error) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := fn(sharedClient, w, r); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
		}
	}
}

//------------------------------------------------------------------------------

func parseID(r *http.Request) (*int64, error) {
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		return nil, err
	}
	id64 := int64(id)
	return &id64, nil
}

func renderTemplate(w http.ResponseWriter, name string, data map[string]interface{}) error {

	// TODO
	if mi := data["model"]; mi != nil {
		if m, isModel := mi.(model.Model); isModel {
			model.ZeroPrivateFields(m)
			data["model"] = m
		}
	}

	modelJSON, _ := json.Marshal(data["model"])
	data["modelJSON"] = string(modelJSON)

	if fields, ok := data["fields"]; ok {
		fieldsJSON, _ := json.Marshal(fields)
		data["fieldsJSON"] = string(fieldsJSON)
	}

	if err := templates[name].ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}
	return nil
}

func unmarshalFormInto(r *http.Request, out interface{}) error {
	if err := r.ParseForm(); err != nil {
		return err
	}
	return json.Unmarshal([]byte(r.PostForm.Get("json_input")), out)
}

//------------------------------------------------------------------------------

func uiRedirect(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/ui", http.StatusFound)
}

//------------------------------------------------------------------------------

func Root(sg *client.Client, w http.ResponseWriter, r *http.Request) error {
	http.Redirect(w, r, "/ui/cloud_accounts", http.StatusFound)
	return nil
}
