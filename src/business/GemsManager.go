package business

import (
	"encoding/json"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"strconv"
	"sync"
)

type GemsManager struct {
	server  *http.Server
	session *sync.Map
}

func NewGemsManager(address string) (*GemsManager, error) {
	gemsManager := new(GemsManager)
	gemsManager.session = new(sync.Map)

	serverMux := http.NewServeMux()
	serverMux.HandleFunc("/login", gemsManager.login)
	serverMux.HandleFunc("/check", gemsManager.check)

	gemsManager.server = &http.Server{
		Addr:    address,
		Handler: serverMux,
	}

	return gemsManager, nil
}

func (m *GemsManager) Serve() error {
	return m.server.ListenAndServe()
}

func returnJSON(w http.ResponseWriter, v interface{}) {
	bytes, e := json.Marshal(v)
	if e != nil {
		panic(e)
	}
	w.Header().Set("Content-Type", "application/json")
	if _, e := w.Write(bytes); e != nil {
		panic(e)
	}
}

func generateSessionID(w http.ResponseWriter) string {
	sessionID := &http.Cookie{
		Name:  "session_id",
		Value: strconv.Itoa(rand.Int()),
	}
	w.Header().Add("Set-Cookie", sessionID.String())
	return sessionID.Value
}

func (m *GemsManager) check(w http.ResponseWriter, r *http.Request) {
	sessionID, e := r.Cookie("session_id")
	if e != nil {
		panic(e)
	}

	value, ok := m.session.Load(sessionID.Value)
	if ok {
		if _, e := w.Write([]byte(value.(string))); e != nil {
			panic(e)
		}
	}
}

func (m *GemsManager) login(w http.ResponseWriter, r *http.Request) {
	// Parse Post JSON Body
	if e := r.ParseForm(); e != nil {
		panic(e)
	}
	form := make(map[string]string)
	body, e := ioutil.ReadAll(r.Body)
	if e != nil {
		panic(e)
	}
	if e := json.Unmarshal(body, &form); e != nil {
		panic(e)
	}

	username := form["username"]
	m.session.Store(generateSessionID(w), username)
	// Return JSON
	result := make(map[string]interface{})
	returnJSON(w, result)
}

func (m *GemsManager) checkServer() (bool, error) {
	connection, e := net.Dial("tcp", "")
	if e != nil {
		return false, e
	}
	return true, connection.Close()
}
