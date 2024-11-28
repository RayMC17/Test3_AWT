package main

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/RayMC17/bookclub-api/internal/data"
	"github.com/RayMC17/bookclub-api/internal/validator"
)

func (a *applicationDependencies) createBookHandler(w http.ResponseWriter, r *http.Request) {
	var incomingData struct {
		Title           string   `json:"title"`
		Authors         []string `json:"authors"`
		ISBN            string   `json:"isbn"`
		PublicationDate string   `json:"publication_date"`
		Genre           string   `json:"genre"`
		Description     string   `json:"description"`
		AverageRating   float64  `json:"average_rating"`
	}
	err := a.readJSON(w, r, &incomingData)
	if err != nil {
		a.badRequestResponse(w, r, err)
		return
	}

	book := &data.Book{
		Title:         incomingData.Title,
		Authors:       incomingData.Authors,
		ISBN:          incomingData.ISBN,
		Genre:         incomingData.Genre,
		Description:   incomingData.Description,
		AverageRating: incomingData.AverageRating,
	}

	v := validator.New()
	data.ValidateBook(v, book)
	if !v.Valid() {
		a.failedValidationResponse(w, r, v.Errors)
		return
	}

	err = a.bookModel.Insert(book)
	if err != nil {
		a.serverErrorResponse(w, r, err)
		return
	}

	headers := make(http.Header)
	headers.Set("Location", fmt.Sprintf("/api/v1/books/%d", book.ID))
	data := envelope{"book": book}
	err = a.writeJSON(w, http.StatusCreated, data, headers)
	if err != nil {
		a.serverErrorResponse(w, r, err)
	}
}

func (a *applicationDependencies) getBookHandler(w http.ResponseWriter, r *http.Request) {
	id, err := a.readIDParam(r)
	if err != nil {
		a.notFoundResponse(w, r)
		return
	}

	book, err := a.bookModel.Get(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			a.notFoundResponse(w, r)
		default:
			a.serverErrorResponse(w, r, err)
		}
		return
	}

	data := envelope{"book": book}
	err = a.writeJSON(w, http.StatusOK, data, nil)
	if err != nil {
		a.serverErrorResponse(w, r, err)
	}
}

func (a *applicationDependencies) updateBookHandler(w http.ResponseWriter, r *http.Request) {
	id, err := a.readIDParam(r)
	if err != nil {
		a.notFoundResponse(w, r)
		return
	}

	book, err := a.bookModel.Get(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			a.notFoundResponse(w, r)
		default:
			a.serverErrorResponse(w, r, err)
		}
		return
	}

	var incomingData struct {
		Title         *string   `json:"title"`
		Authors       *[]string `json:"authors"`
		ISBN          *string   `json:"isbn"`
		Genre         *string   `json:"genre"`
		Description   *string   `json:"description"`
		AverageRating *float64  `json:"average_rating"`
	}
	err = a.readJSON(w, r, &incomingData)
	if err != nil {
		a.badRequestResponse(w, r, err)
		return
	}

	if incomingData.Title != nil {
		book.Title = *incomingData.Title
	}
	if incomingData.Authors != nil {
		book.Authors = *incomingData.Authors
	}
	if incomingData.ISBN != nil {
		book.ISBN = *incomingData.ISBN
	}
	if incomingData.Genre != nil {
		book.Genre = *incomingData.Genre
	}
	if incomingData.Description != nil {
		book.Description = *incomingData.Description
	}
	if incomingData.AverageRating != nil {
		book.AverageRating = *incomingData.AverageRating
	}

	v := validator.New()
	data.ValidateBook(v, book)
	if !v.Valid() {
		a.failedValidationResponse(w, r, v.Errors)
		return
	}

	err = a.bookModel.Update(book)
	if err != nil {
		a.serverErrorResponse(w, r, err)
		return
	}

	data := envelope{"book": book}
	err = a.writeJSON(w, http.StatusOK, data, nil)
	if err != nil {
		a.serverErrorResponse(w, r, err)
	}
}

func (a *applicationDependencies) deleteBookHandler(w http.ResponseWriter, r *http.Request) {
	id, err := a.readIDParam(r)
	if err != nil {
		a.notFoundResponse(w, r)
		return
	}

	err = a.bookModel.Delete(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			a.notFoundResponse(w, r)
		default:
			a.serverErrorResponse(w, r, err)
		}
		return
	}

	data := envelope{"message": "book successfully deleted"}
	err = a.writeJSON(w, http.StatusOK, data, nil)
	if err != nil {
		a.serverErrorResponse(w, r, err)
	}
}

func (a *applicationDependencies) listBooksHandler(w http.ResponseWriter, r *http.Request) {
	var queryParametersData struct {
		data.Filters
	}
	queryParameters := r.URL.Query()

	v := validator.New()

	queryParametersData.Filters.Page = a.getSingleIntegerParameter(queryParameters, "page", 1, v)
	queryParametersData.Filters.PageSize = a.getSingleIntegerParameter(queryParameters, "page_size", 10, v)
	queryParametersData.Filters.Sort = a.getSingleQueryParameter(queryParameters, "sort", "id")
	queryParametersData.Filters.SortSafelist = []string{"id", "title", "-id", "-title"}

	// Check if our filters are valid
	data.ValidateFilters(v, &queryParametersData.Filters)
	if !v.Valid() {
		a.failedValidationResponse(w, r, v.Errors)
		return
	}

	books, metadata, err := a.bookModel.GetAll(
		queryParametersData.Filters,
	)
	if err != nil {
		a.serverErrorResponse(w, r, err)
		return
	}

	responseData := envelope{
		"books":     books,
		"@metadata": metadata,
	}
	err = a.writeJSON(w, http.StatusOK, responseData, nil)
	if err != nil {
		a.serverErrorResponse(w, r, err)
	}
}

func (a *applicationDependencies) searchBookHandler(w http.ResponseWriter, r *http.Request) {
	var queryParametersData struct {
		Title  string
		Author string
		Genre  string
		data.Filters
	}
	queryParameters := r.URL.Query()
	// Load the query parameters into our struct
	queryParametersData.Title = a.getSingleQueryParameter(queryParameters, "title", "")
	queryParametersData.Genre = a.getSingleQueryParameter(queryParameters, "genre", "")
	queryParametersData.Author = a.getSingleQueryParameter(queryParameters, "author", "")
	v := validator.New()

	queryParametersData.Filters.Page = a.getSingleIntegerParameter(queryParameters, "page", 1, v)
	queryParametersData.Filters.PageSize = a.getSingleIntegerParameter(queryParameters, "page_size", 10, v)
	queryParametersData.Filters.Sort = a.getSingleQueryParameter(queryParameters, "sort", "id")
	queryParametersData.Filters.SortSafelist = []string{"id", "title", "genre", "author", "-id", "-title", "-genre", "-author"}

	// Check if our filters are valid
	data.ValidateFilters(v, &queryParametersData.Filters)
	if !v.Valid() {
		a.failedValidationResponse(w, r, v.Errors)
		return
	}

	books, metadata, err := a.bookModel.GetAllFilters(
		queryParametersData.Title,
		queryParametersData.Genre,
		queryParametersData.Author,
		queryParametersData.Filters,
	)
	if err != nil {
		a.serverErrorResponse(w, r, err)
		return
	}

	responseData := envelope{
		"books":     books,
		"@metadata": metadata,
	}
	err = a.writeJSON(w, http.StatusOK, responseData, nil)
	if err != nil {
		a.serverErrorResponse(w, r, err)
	}

}
