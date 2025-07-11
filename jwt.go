// Copyright 2024 C Delezenski <cmd184psu@gmail.com>
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//      http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package alfredo

import (
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
)

const (
	DefaultUserCredsConfig = "usercreds.conf"
	ContentType            = "Content-Type"
	ApplicationJson        = "application/json"
	LoginRoute             = "/login"
	LogoutRoute            = "/logout"
	StaticRoute            = "/*"
	//StaticDirRoute         = "./static"
	ExpireTime = 120 //expire JWT token in 120 minutes
)

func ContentTypeJSON() (string, string) {
	return ContentType, ApplicationJson
}

func GenerateJWTKey() string {
	// 32 bytes * 8 bits/byte = 256 bits
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		log.Fatal(err)
	}

	// Encode key to base64 to make it suitable for JWT
	return base64.RawURLEncoding.EncodeToString(key)
}

type JwtCredentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
	PassCode string `json:"passcode"`
}

type JwtClaims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

type JwtHttpsServerStruct struct {
	jwtKey         []byte
	Port           int
	Router         *chi.Mux
	publickey      string
	privatekey     string
	pathMap        map[string]bool
	secure         bool
	srv            *http.Server
	StaticDirRoute string
}

func (jhs *JwtHttpsServerStruct) Init(port int) {
	//Router = chi.NewRouter
	jhs.secure = false
	jhs.Router = chi.NewRouter()
	jhs.Router.Use(middleware.Logger)

	jhs.SetKey([]byte(GenerateJWTKey()))
	jhs.SetPort(port)

}

func (jhs *JwtHttpsServerStruct) SetPort(p int) {
	jhs.Port = p
}
func (jhs JwtHttpsServerStruct) WithPort(p int) JwtHttpsServerStruct {
	jhs.Port = p
	return jhs
}
func (jhs JwtHttpsServerStruct) GetPort() int {
	return jhs.Port
}
func (jhs *JwtHttpsServerStruct) SetStaticDirRoute(dir string) {
	jhs.StaticDirRoute = dir
}
func (jhs JwtHttpsServerStruct) WithStaticDirRoute(dir string) JwtHttpsServerStruct {
	jhs.StaticDirRoute = dir
	return jhs
}
func (jhs JwtHttpsServerStruct) GetStaticDirRoute() string {
	return jhs.StaticDirRoute
}

func (jhs *JwtHttpsServerStruct) SetKey(k []byte) {
	jhs.jwtKey = k
}
func (jhs JwtHttpsServerStruct) WithKey(k []byte) JwtHttpsServerStruct {
	jhs.jwtKey = k
	return jhs
}
func (jhs JwtHttpsServerStruct) GetKey() []byte {
	return jhs.jwtKey
}

func (jhs *JwtHttpsServerStruct) AcquireKey(f string) {
	if len(os.Getenv("JWT_KEY")) != 0 {
		jhs.jwtKey = []byte(os.Getenv("JWT_KEY"))
		return
	}
	if FileExistsEasy(f) {
		key, err := os.ReadFile(f)
		if err != nil {
			log.Fatal("Error loading JWT key: ", err)
		}
		jhs.SetKey(key)
		return
	}
	jhs.SetKey([]byte(GenerateJWTKey()))
}

func (jhs JwtHttpsServerStruct) GetCertFiles() (string, string) {
	return jhs.privatekey, jhs.publickey
}
func (jhs *JwtHttpsServerStruct) SetCertFiles(priv, pub string) {
	jhs.privatekey = priv
	jhs.publickey = pub
	jhs.secure = true
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (jhs JwtHttpsServerStruct) RouteExists(r string) bool {
	return jhs.pathMap[r]
}

//	func (jhs *JwtHttpsServerStruct) SetLoginHandler(h http.Handler) {
//		r = chi.NewRouter()
//		r.Post("/login", h)
//		return r
//		// if !jhs.pathMap[LoginRoute] {
//		// 	jhs.Router.Post(LoginRoute, h)
//		// }
//	}
func (jhs *JwtHttpsServerStruct) SetupStaticRoutes(index string) {
	fs := http.FileServer(http.Dir("./static/js"))
	jhs.Router.Handle("/js/*", http.StripPrefix("/js/", fs))
	jhs.Router.Handle("/css/*", http.StripPrefix("/css/", http.FileServer(http.Dir("./static/css"))))
	jhs.Router.Handle("/images/*", http.StripPrefix("/images/", http.FileServer(http.Dir("./static/images"))))
	jhs.Router.Handle("/webfonts/*", http.StripPrefix("/webfonts/", http.FileServer(http.Dir("./static/webfonts"))))
	log.Printf("serving index file: %s\n", index)
	jhs.Router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("in / route")
		log.Printf("URI=%s", r.RequestURI)
		log.Printf("serving index file: %s\n", index)
		http.ServeFile(w, r, "./static/"+index)
	})
}

// func (jhs *JwtHttpsServerStruct) SetupStaticRoutes(index string) {
// 	fs := http.FileServer(http.Dir("./static/js"))
// 	r.PathPrefix("/js/").Handler(http.StripPrefix("/js/", fs))
// 	r.PathPrefix("/css/").Handler(http.StripPrefix("/css/", http.FileServer(http.Dir("./static/css"))))
// 	r.PathPrefix("/webfonts/").Handler(http.StripPrefix("/webfonts/", http.FileServer(http.Dir("./static/webfonts"))))
// 	log.Printf("serving index file: %s\n", index)
// 	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
// 		log.Printf("in / route")
// 		log.Printf("URI=%s", r.RequestURI)
// 		log.Printf("serving index file: %s\n", index)
// 		http.ServeFile(w, r, "./static/"+index)
// 	})

// }

func (jhs *JwtHttpsServerStruct) StartServer() error {
	jhs.pathMap = make(map[string]bool)

	chi.Walk(jhs.Router, func(method string, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		jhs.pathMap[route] = true
		return nil
	})

	// if !jhs.pathMap[LoginRoute] {
	// 	jhs.Router.Post(LoginRoute, loginHandler)
	// }

	if !jhs.pathMap[LogoutRoute] {
		jhs.Router.Post(LogoutRoute, logoutHandler)
	}
	// Serve static files
	if !jhs.pathMap[StaticRoute] {
		jhs.Router.Handle(StaticRoute, http.FileServer(http.Dir(jhs.GetStaticDirRoute())))
	}
	if !jhs.pathMap["/"] {
		jhs.Router.Handle(StaticRoute, http.FileServer(http.Dir(jhs.GetStaticDirRoute())))
	}

	var err error
	// Load self-signed SSL certificate
	if jhs.secure {
		cert, err := tls.LoadX509KeyPair(jhs.GetCertFiles())
		if err != nil {
			log.Fatal("Error loading SSL certificate: ", err)
			return err
		}

		jhs.srv = &http.Server{
			Addr:      fmt.Sprintf(":%d", jhs.GetPort()),
			Handler:   jhs.Router,
			TLSConfig: &tls.Config{Certificates: []tls.Certificate{cert}},
		}
		log.Printf("--- SSL Enabled ---")
		log.Printf("Starting SSL server on :%d", jhs.GetPort())
		err = jhs.srv.ListenAndServeTLS("", "")
		if err != nil {
			log.Fatal("Server failed: ", err)
			return err
		}
	} else {

		jhs.srv = &http.Server{
			Addr:    fmt.Sprintf(":%d", jhs.GetPort()),
			Handler: jhs.Router,
		}
		log.Printf("Starting server on :%d", jhs.GetPort())
		err = jhs.srv.ListenAndServe()
		if err != nil {
			log.Fatal("Server failed: ", err)
			return err
		}
	}
	return nil
}

func (jhs *JwtHttpsServerStruct) UpdateClaims(username string, w http.ResponseWriter) {
	claims := &JwtClaims{
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ExpireTime * time.Minute)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jhs.GetKey())
	if err != nil {
		log.Printf("Error signing token: %v", err) // Log the error
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"token": tokenString})

}

func (jhs *JwtHttpsServerStruct) AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			log.Println("err1")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		bearerToken := strings.Split(authHeader, " ")
		if len(bearerToken) != 2 {
			log.Println("err2")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		token, err := jwt.ParseWithClaims(bearerToken[1], &JwtClaims{}, func(token *jwt.Token) (interface{}, error) {
			//return jwtKey, nil
			return jhs.GetKey(), nil
		})

		if err != nil {
			log.Printf("err3: %s", err.Error())
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if claims, ok := token.Claims.(*JwtClaims); ok && token.Valid {
			r.Header.Set("Username", claims.Username)
			next.ServeHTTP(w, r)
		} else {
			log.Println("err4")
			w.WriteHeader(http.StatusUnauthorized)
		}
	}
}

func (jhs JwtHttpsServerStruct) ValidateBearerToken(w http.ResponseWriter, r *http.Request) bool {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return false
	}

	// Extract Bearer token
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		w.WriteHeader(http.StatusUnauthorized)
		return false
	}

	return strings.EqualFold(parts[1], string(jhs.GetKey()))
}

// IsTokenExpired checks if a JWT token has expired on the client side
// Returns true if the token is expired, false if still valid, and error if token is malformed
func IsTokenExpired(tokenString string) (bool, error) {
        if len(tokenString) == 0 {
                return true, nil
        }
        // Parse the token without verification (client-side check only)
        token, _, err := new(jwt.Parser).ParseUnverified(tokenString, &JwtClaims{})
        if err != nil {
                return true, fmt.Errorf("failed to parse token: %v", err)
        }

        if claims, ok := token.Claims.(*JwtClaims); ok {
                // Check if token has expired
                if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(time.Now()) {
                        return true, nil // Token is expired
                }
                return false, nil // Token is still valid
        }

        return true, fmt.Errorf("invalid token claims")
}

func IsTokenExpiredEasy(tokenString string) bool {
        b, err := IsTokenExpired(tokenString)
        if err != nil {
                panic("IsTokenExpiredEasy: " + err.Error())
        }
        return b
}

