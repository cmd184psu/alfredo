package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/cmd184psu/alfredo"
)

type SampleData struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Value string `json:"value"`
}

var jserver alfredo.JwtHttpsServerStruct

func main() {
	//initialize
	jserver.Init(5443)
	//	jserver.AcquireKey("jwt.key")

	//handlers
	jserver.Router.Post(alfredo.LoginRoute, loginHandler)
	//	jserver.Router.Get("/data", jserver.AuthMiddleware(GetData))
	jserver.Router.Get("/data", jserver.AuthMiddleware(GetData))

	// slice := jserver.Router.Routes()
	// for i := 0; i < len(slice); i++ {
	// 	fmt.Println(slice[i])
	// }
	//start server and wait
	if err := jserver.StartServer(); err != nil {
		panic(err.Error())
	}
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	var creds alfredo.JwtCredentials
	err := json.NewDecoder(r.Body).Decode(&creds)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	log.Printf("Attempting to authenticate user: %s:%s\n", creds.Username, creds.Password)
	if alfredo.FileAuthenticate(creds.Username, creds.Password, alfredo.DefaultUserCredsConfig) {
		log.Printf("authentication was successful")
		jserver.UpdateClaims(creds.Username, w)
		//jserver.UpdateCookie(creds.Username, w)
	} else {
		log.Printf("authentication failed")
		w.WriteHeader(http.StatusUnauthorized)
	}
}

func GetData(w http.ResponseWriter, r *http.Request) {
	sampleData := []SampleData{
		{ID: 1, Name: "Item 1", Value: "Value 1"},
		{ID: 2, Name: "Item 2", Value: "Value 2"},
		{ID: 3, Name: "Item 3", Value: "Value 3"},
	}

	w.Header().Set(alfredo.ContentTypeJSON())
	json.NewEncoder(w).Encode(sampleData)
}
