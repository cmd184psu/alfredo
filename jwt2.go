// Copyright 2025 C Delezenski <cmd184psu@gmail.com>
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
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type ServerConfig struct {
	Passcode string
	TokenTTL time.Duration
	CertFile string
	KeyFile  string
	Insecure bool
	Address  string
	Port     int
	jwtKey   []byte // JWT signing key

	// RolesPath string
}

type ServerBuilder struct {
	config    ServerConfig
	logger    *log.Logger
	staticDir string // Default static directory
}

type LoginResponse struct {
	Token string `json:"token"`
}

func NewServerBuilder() *ServerBuilder {
	return &ServerBuilder{
		config: ServerConfig{
			TokenTTL: time.Minute,
			Address:  "localhost",
			Port:     3000,
		},
	}
}

func (b *ServerBuilder) WithPasscode(passcode string) *ServerBuilder {
	b.config.Passcode = passcode
	return b
}

func (b *ServerBuilder) WithTokenTTL(minutes int) *ServerBuilder {
	b.config.TokenTTL = time.Duration(minutes) * time.Minute
	return b
}

func (b *ServerBuilder) WithSSL(cert, key string) *ServerBuilder {
	b.config.CertFile = cert
	b.config.KeyFile = key
	return b
}

func (b *ServerBuilder) WithAddress(addr string, port int) *ServerBuilder {
	b.config.Address = addr
	b.config.Port = port
	return b
}

func (b *ServerBuilder) WithLogger(logger *log.Logger) *ServerBuilder {
	b.logger = logger
	return b
}

// func (b *ServerBuilder) WithRolesFile(path string) *ServerBuilder {
// 	b.config.RolesPath = path
// 	return b
// }

func (b *ServerBuilder) WithStaticDir(dir string) *ServerBuilder {
	if dir == "" {
		dir = "./static"
	}
	b.staticDir = dir
	return b
}

func (b *ServerBuilder) WithJWTKey(keyPath string) *ServerBuilder {
	if len(os.Getenv("JWT_KEY")) != 0 {
		b.config.jwtKey = []byte(os.Getenv("JWT_KEY"))
		return b
	}
	if FileExistsEasy(keyPath) {
		key, err := os.ReadFile(keyPath)
		if err != nil {
			log.Fatal("Error loading JWT key: ", err)
		}
		b.config.jwtKey = key
		return b
	}

	b.config.jwtKey = []byte(GenerateJWTKey())
	return b
}

func (b *ServerBuilder) WithCertificateFiles(cert, key string) *ServerBuilder {
	if cert != "" && key != "" {
		b.config.CertFile = cert
		b.config.KeyFile = key
	} else {
		b.config.CertFile = ""
		b.config.KeyFile = ""
	}
	return b
}
func (b *ServerBuilder) Build() *JWTServer {
	return newJWTServer(b)
}

type Middleware func(http.HandlerFunc) http.HandlerFunc

func Chain(handler http.HandlerFunc, middlewares ...Middleware) http.HandlerFunc {

	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return handler
}

func (s *JWTServer) JWTMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tokenString := extractBearerToken(r.Header.Get("Authorization"))
		if tokenString == "" {
			http.Error(w, "Unauthorized - no token", http.StatusUnauthorized)
			return
		}

		s.mu.Lock()
		_, blacklisted := s.blacklist[tokenString]
		s.mu.Unlock()
		if blacklisted {
			http.Error(w, "Unauthorized - token revoked", http.StatusUnauthorized)
			return
		}

		token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
			return s.privateKey, nil
		})
		if err != nil || !token.Valid {
			http.Error(w, "Unauthorized - invalid token", http.StatusUnauthorized)
			return
		}

		next(w, r)
	}
}
func (s *JWTServer) NoopMiddleware() Middleware {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			next(w, r)
		}
	}
}
func (s *JWTServer) MiddleWareRequireRole(role string) Middleware {
	log.Println("MiddlewareRequireRole called with role:", role)
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			tokenStr := extractBearerToken(r.Header.Get("Authorization"))
			token, _ := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
				return s.privateKey, nil
			})

			//claims2, ok2 := token.Claims.(jwt.MapClaims)
			//log.Printf("\t%s \nok=%t\n", PrettyPrint(claims2), ok2)
			if _, ok := token.Claims.(jwt.MapClaims); ok { //&& claims["role"] == role {
				log.Printf("about to execute next(w,r)\n")
				next(w, r)
			} else {
				http.Error(w, "Forbidden - insufficient role", http.StatusForbidden)
			}
		}
	}
}

func MiddleWareWithLogging(logger *log.Logger) Middleware {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			logger.Printf("%s %s", r.Method, r.URL.Path)
			next(w, r)
		}
	}
}

func extractBearerToken(header string) string {
	if strings.HasPrefix(header, "Bearer ") {
		return strings.TrimPrefix(header, "Bearer ")
	}
	return ""
}

var defaultRoles = map[string]string{
	"letmein":  "admin",
	"readonly": "viewer",
	"guest":    "viewer",
}

type JWTServer struct {
	config     ServerConfig
	privateKey []byte
	router     *http.ServeMux
	logger     *log.Logger
	mu         sync.Mutex
	blacklist  map[string]struct{}
	roles      map[string]string
	staticDir  string // Default static directory
}

func newJWTServer(b *ServerBuilder) *JWTServer {
	logger := b.logger
	if logger == nil {
		logger = log.New(os.Stdout, "[jwtserver] ", log.LstdFlags)
	}

	s := &JWTServer{
		config:     b.config,
		privateKey: b.config.jwtKey,
		router:     http.NewServeMux(),
		blacklist:  make(map[string]struct{}),
		logger:     logger,
		//		roles:      loadRoles(b.config.RolesPath),
		roles: defaultRoles,
	}
	if b.staticDir == "" {
		b.staticDir = "./static"
	}
	s.routes()
	//s.ServeStatic("/*", b.staticDir)
	s.ServeStaticDirectory("/", b.staticDir)
	return s
}

func (s *JWTServer) Start() error {
	addr := s.config.Address + ":" + strconv.Itoa(s.config.Port)
	s.logger.Printf("Starting server at %s", addr)

	if s.config.CertFile != "" && s.config.KeyFile != "" {
		return http.ListenAndServeTLS(addr, s.config.CertFile, s.config.KeyFile, s.router)
	}
	return http.ListenAndServe(addr, s.router)
}

func (s *JWTServer) routes() {
	s.router.HandleFunc("/login", s.handleLogin())
	s.router.HandleFunc("/logout", s.JWTMiddleware(s.handleLogout()))
	s.router.HandleFunc("/refresh", s.JWTMiddleware(s.handleRefresh()))
}

func (s *JWTServer) AddRoute(pattern string, handler http.HandlerFunc, protected bool, middlewares ...Middleware) {
	if protected {
		middlewares = append([]Middleware{s.JWTMiddleware}, middlewares...)
	}
	s.router.HandleFunc(pattern, Chain(handler, middlewares...))
}

func (s *JWTServer) ServeStaticDirectory(pattern string, dir string) {
	if len(dir) == 0 {
		dir = s.staticDir
	}
	if !strings.HasSuffix(pattern, "/") {
		pattern += "/"
	}
	log.Printf("Serving static files from %s at %s", dir, pattern)
	fs := http.FileServer(http.Dir(dir))
	s.router.Handle(pattern, http.StripPrefix(pattern, fs))
}

//	func (s *JWTServer) ServeStaticDirectory(pattern string, dir string) {
//		if len(dir) == 0 {
//			dir = s.staticDir
//		}
//		log.Printf("Serving static files from %s at %s", dir, pattern)
//		fs := http.FileServer(http.Dir(dir))
//		s.router.Handle(pattern, http.StripPrefix(pattern, fs))
//	}
func (s *JWTServer) ServeStaticFile(pattern string, fileName string) {
	log.Printf("Serving static file %s at %s", fileName, pattern)
	s.router.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, fileName)
	})
}

func (s *JWTServer) handleLogin() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Passcode string `json:"passcode"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		if req.Passcode != s.config.Passcode {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		role := s.roles[req.Passcode]
		if role == "" {
			role = "viewer"
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"exp":  time.Now().Add(s.config.TokenTTL).Unix(),
			"iat":  time.Now().Unix(),
			"sub":  req.Passcode,
			"role": role,
		})

		signedToken, err := token.SignedString(s.privateKey)
		if err != nil {
			http.Error(w, "Could not sign token", http.StatusInternalServerError)
			return
		}

		resp := map[string]string{"token": signedToken}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

func (s *JWTServer) handleLogout() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := extractBearerToken(r.Header.Get("Authorization"))
		if token == "" {
			http.Error(w, "Missing token", http.StatusBadRequest)
			return
		}
		s.mu.Lock()
		s.blacklist[token] = struct{}{}
		s.mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}
}

func (s *JWTServer) handleRefresh() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tokenString := extractBearerToken(r.Header.Get("Authorization"))
		token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
			return s.privateKey, nil
		})
		if err != nil || !token.Valid {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			http.Error(w, "Invalid token claims", http.StatusBadRequest)
			return
		}

		newToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"exp":  time.Now().Add(s.config.TokenTTL).Unix(),
			"iat":  time.Now().Unix(),
			"sub":  claims["sub"],
			"role": claims["role"],
		})

		signedToken, err := newToken.SignedString(s.privateKey)
		if err != nil {
			http.Error(w, "Token refresh failed", http.StatusInternalServerError)
			return
		}

		resp := map[string]string{"token": signedToken}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

// func loadRoles(path string) map[string]string {
// 	file, err := os.ReadFile(path)
// 	if err != nil {
// 		log.Printf("Warning: could not load roles from %s: %v", path, err)
// 		return map[string]string{}
// 	}
// 	var roles map[string]string
// 	if err := json.Unmarshal(file, &roles); err != nil {
// 		log.Printf("Warning: invalid roles.json: %v", err)
// 		return map[string]string{}
// 	}
// 	return roles
// }
