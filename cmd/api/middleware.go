package main

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/RayMC17/bookclub-api/internal/data"
	"github.com/RayMC17/bookclub-api/internal/validator"
	"golang.org/x/time/rate"
)

type client struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}


 
//This code ends here

// Middleware: Panic Recovery
func (a *applicationDependencies) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				w.Header().Set("Connection", "close")
				a.serverErrorResponse(w, r, fmt.Errorf("%s", err))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// Middleware: Rate Limiting
func (a *applicationDependencies) rateLimit(next http.Handler) http.Handler {
	clients := make(map[string]*client)
	var mu sync.Mutex

	go func() {
		for {
			time.Sleep(time.Minute)
			mu.Lock()
			for ip, c := range clients {
				if time.Since(c.lastSeen) > 3*time.Minute {
					delete(clients, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !a.config.limiter.enabled {
			next.ServeHTTP(w, r)
			return
		}

		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			a.serverErrorResponse(w, r, err)
			return
		}

		mu.Lock()
		if _, found := clients[ip]; !found {
			clients[ip] = &client{
				limiter: rate.NewLimiter(rate.Limit(a.config.limiter.rps), a.config.limiter.burst),
			}
		}

		clients[ip].lastSeen = time.Now()
		if !clients[ip].limiter.Allow() {
			mu.Unlock()
			a.rateLimitExceededResponse(w, r)
			return
		}
		mu.Unlock()

		next.ServeHTTP(w, r)
	})
}

// Middleware: Logging
func (a *applicationDependencies) logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		a.logger.Info("started request", "method", r.Method, "url", r.URL.String())
		next.ServeHTTP(w, r)
		a.logger.Info("completed request", "method", r.Method, "url", r.URL.String(), "duration", time.Since(start))
	})
}

// Middleware: Authentication (Stub)
func (a *applicationDependencies) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		/*This header tells the servers not to cache the response when
		the Authorization header changes. This also means that the server is not
		supposed to serve the same cached data to all users regardless of their
		Authorization values. Each unique user gets their own cache entry*/
		w.Header().Add("Vary", "Authorization")

		/*Get the Authorization Header from the request. It should have the Bearer token*/
		authorizationHeader := r.Header.Get("Authorization")

		//if no authorization header found, then its an anonymous user
		if authorizationHeader == "" {

			r = a.contextSetUser(r, data.AnonymouseUser)
			next.ServeHTTP(w, r)
			return
		}
		/* Bearer token present so parse it. The Bearer token is in the form
		Authorization: Bearer IEYZQUBEMPPAKPOAWTPV6YJ6RM
		We will implement invalidAuthenticationTokenResponse() later */

		headerParts := strings.Split(authorizationHeader, " ")
		if len(headerParts) != 2 || headerParts[0] != "Bearer" {
			a.invalidAuthenticationTokenResponse(w, r)
			return
		}
		//get the actual token
		token := headerParts[1]
		//validatte
		v := validator.New()

		data.ValidatetokenPlaintext(v, token)
		if !v.Valid() {

			a.invalidAuthenticationTokenResponse(w, r)
			return
		}

		//get the user info relatedw with this authentication token
		user, err := a.userModel.GetForToken(data.ScopeAuthentication, token)
		if err != nil {
			println("adfadsfadsf")
			switch {
			case errors.Is(err, data.ErrRecordNotFound):
				a.invalidAuthenticationTokenResponse(w, r)
			default:
				a.serverErrorResponse(w, r, err)
			}
			return
		}
		//add the retrieved user info to the context
		r = a.contextSetUser(r, user)
		//call the next handler in the chair
		next.ServeHTTP(w, r)
	})
}

// Unauthorized Response Helper
func (a *applicationDependencies) unauthorizedResponse(w http.ResponseWriter, r *http.Request) {
	message := "You are not authorized to access this resource"
	a.errorResponseJSON(w, r, http.StatusUnauthorized, message)
}

func (a *applicationDependencies) requireAuthenticatedUser(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := a.contextGetUser(r)

		if user.IsAnonymous() {
			a.invalidAuthenticationTokenResponse(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// check if user is activated
func (a *applicationDependencies) requireActivatedUser(next http.HandlerFunc) http.HandlerFunc {
	fn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := a.contextGetUser(r)

		if !user.Activated {
			a.inactiveAccountResponse(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})

	//We pass the activation check middleware to the authentication
	// middleware to call (next) if the authentication check succeeds
	// In other words, only check if the user is activated if they are
	// actually authenticated.
	return a.requireAuthenticatedUser(fn)
}

//Step 1: Handles both simple and preflight CORS Request

func (a *applicationDependencies) enableCORS(next http.Handler) http.Handler {                             
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
 
		 w.Header().Add("Vary", "Origin")
		 w.Header().Add("Vary", "Access-Control-Request-Method")
		origin := r.Header.Get("Origin")

		if origin != "" {
			for i:= range a.config.cors.trustedOrigins {
				if origin == a.config.cors.trustedOrigins[i] {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					// check if it is a Preflight CORS request
						if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
						w.Header().Set("Access-Control-Allow-Method", "OPTIONS, PUT, PATCH, POST, DELETE")
		 				w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
						w.WriteHeader(http.StatusOK)
             		 	return
          			}

					break
				}
			}
		}

		 next.ServeHTTP(w, r)
	 })
 }
