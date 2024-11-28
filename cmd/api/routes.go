package main

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func (a *applicationDependencies) routes() http.Handler {
	router := httprouter.New()

	// Health check route
	router.HandlerFunc(http.MethodGet, "/api/v1/healthcheck", a.requireActivatedUser(a.healthCheckHandler))

	// Books routes
	router.HandlerFunc(http.MethodGet, "/api/v1/books", a.requireActivatedUser(a.listBooksHandler))         //done
	router.HandlerFunc(http.MethodGet, "/api/v1/books/:id", a.requireActivatedUser(a.getBookHandler))       //done
	router.HandlerFunc(http.MethodPost, "/api/v1/books", a.requireActivatedUser(a.createBookHandler))       //done
	router.HandlerFunc(http.MethodPut, "/api/v1/books/:id", a.requireActivatedUser(a.updateBookHandler))    //done
	router.HandlerFunc(http.MethodDelete, "/api/v1/books/:id", a.requireActivatedUser(a.deleteBookHandler)) //done
	router.HandlerFunc(http.MethodGet, "/api/v1/book/search", a.requireActivatedUser(a.searchBookHandler))  //done

	// Reading Lists routes
	router.HandlerFunc(http.MethodGet, "/api/v1/lists", a.requireActivatedUser(a.listReadingListsHandler))                       //done
	router.HandlerFunc(http.MethodGet, "/api/v1/lists/:id", a.requireActivatedUser(a.getReadingListHandler))                     //done
	router.HandlerFunc(http.MethodPost, "/api/v1/lists", a.requireActivatedUser(a.createReadingListHandler))                     //done
	router.HandlerFunc(http.MethodPut, "/api/v1/lists/:id", a.requireActivatedUser(a.updateReadingListHandler))                  //done
	router.HandlerFunc(http.MethodDelete, "/api/v1/lists/:id", a.requireActivatedUser(a.deleteReadingListHandler))               //done
	router.HandlerFunc(http.MethodPost, "/api/v1/lists/:id/books", a.requireActivatedUser(a.addBookToReadingListHandler))        //done
	router.HandlerFunc(http.MethodDelete, "/api/v1/lists/:id/books", a.requireActivatedUser(a.removeBookFromReadingListHandler)) //done

	// Reviews routes
	router.HandlerFunc(http.MethodGet, "/api/v1/books/:id/reviews", a.requireActivatedUser(a.listReviewsHandler))   //done
	router.HandlerFunc(http.MethodPost, "/api/v1/books/:id/reviews", a.requireActivatedUser(a.createReviewHandler)) //done
	router.HandlerFunc(http.MethodPut, "/api/v1/reviews/:id", a.requireActivatedUser(a.updateReviewHandler))        //done
	router.HandlerFunc(http.MethodDelete, "/api/v1/reviews/:id", a.requireActivatedUser(a.deleteReviewHandler))     //done

	// Users routes
	router.HandlerFunc(http.MethodPost, "/api/v1/user", a.makeUserProfileHandler)                                       //done
	router.HandlerFunc(http.MethodGet, "/api/v1/users/:id", a.requireActivatedUser(a.getUserProfileHandler))            //done
	router.HandlerFunc(http.MethodPut, "/api/v1/users/activate", a.activateUserhandler)                                 //done
	router.HandlerFunc(http.MethodPost, "/api/v1/authentocate/token", a.authenticateTokenHandler)                       //done
	router.HandlerFunc(http.MethodGet, "/api/v1/users/:id/lists", a.requireActivatedUser(a.getUserReadingListsHandler)) //done
	router.HandlerFunc(http.MethodGet, "/api/v1/users/:id/reviews", a.requireActivatedUser(a.getUserReviewsHandler))    //done

	// Wrap the entire router with global middleware
	//return a.logRequest(a.rateLimit(a.recoverPanic(router)))
	return a.recoverPanic(a.rateLimit(a.authenticate(router)))
}
