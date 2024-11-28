package main

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/RayMC17/bookclub-api/internal/data"
	"github.com/RayMC17/bookclub-api/internal/validator"
)

// listReadingListsHandler retrieves a list of reading lists.
func (a *applicationDependencies) listReadingListsHandler(w http.ResponseWriter, r *http.Request) {
	var filters data.Filters

	// Set up query parameters for pagination and sorting
	v := validator.New()
	filters.Page = a.getSingleIntegerParameter(r.URL.Query(), "page", 1, v)
	filters.PageSize = a.getSingleIntegerParameter(r.URL.Query(), "page_size", 10, v)
	filters.Sort = a.getSingleQueryParameter(r.URL.Query(), "sort", "id")
	filters.SortSafelist = []string{"id", "name", "-id", "-name"} // Define allowed sort fields

	// Validate filters
	data.ValidateFilters(v, &filters)
	if !v.Valid() {
		a.failedValidationResponse(w, r, v.Errors)
		return
	}

	// Retrieve reading lists from the database
	lists, metadata, err := a.readingListModel.GetAll(filters)
	if err != nil {
		a.serverErrorResponse(w, r, err)
		return
	}

	// Send response with lists and metadata
	response := envelope{
		"reading_lists": lists,
		"metadata":      metadata,
	}
	err = a.writeJSON(w, http.StatusOK, response, nil)
	if err != nil {
		a.serverErrorResponse(w, r, err)
	}
}

func (a *applicationDependencies) getReadingListHandler(w http.ResponseWriter, r *http.Request) {
	// Extract the reading list ID from the URL parameters
	idParam, err := a.readIDParam(r)
	if err != nil || idParam < 1 {
		a.notFoundResponse(w, r)
		return
	}

	//println(idParam)
	// Fetch the reading list from the database
	readingList, err := a.readingListModel.Get(idParam)
	if err != nil {
		if errors.Is(err, data.ErrRecordNotFound) {
			a.notFoundResponse(w, r)
		} else {
			a.serverErrorResponse(w, r, err)
		}
		return
	}

	// Send the reading list in the response
	err = a.writeJSON(w, http.StatusOK, envelope{"reading_list": readingList}, nil)
	if err != nil {
		a.serverErrorResponse(w, r, err)
	}
}

func (a *applicationDependencies) createReadingListHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Created_by  int    `json:"created_by"`
	}

	// Decode JSON body
	err := a.readJSON(w, r, &input)
	if err != nil {
		a.badRequestResponse(w, r, err)
		return
	}

	readingList := &data.ReadingList{
		Name:        input.Name,
		Description: input.Description,
		CreatedBy:   input.Created_by,
	}

	// Validate the input
	v := validator.New()
	data.ValidateReadingList(v, readingList)
	if !v.Valid() {
		a.failedValidationResponse(w, r, v.Errors)
		return
	}

	err = a.userModel.UserExists(input.Created_by)
	if err != nil {
		a.notFoundResponse(w, r)
		return
	}

	// Insert the reading list into the database
	err = a.readingListModel.CreateReadingList(readingList)
	if err != nil {
		a.serverErrorResponse(w, r, err)
		return
	}

	headers := make(http.Header)
	headers.Set("Location", "/api/v1/lists/"+strconv.Itoa(readingList.ID))

	// Send the created reading list in the response
	err = a.writeJSON(w, http.StatusCreated, envelope{"reading_list": readingList}, headers)
	if err != nil {
		a.serverErrorResponse(w, r, err)
	}
}

func (a *applicationDependencies) updateReadingListHandler(w http.ResponseWriter, r *http.Request) {
	// Extract the reading list ID from the URL parameters
	id, err := a.readIDParam(r)
	if err != nil {
		a.notFoundResponse(w, r)
		return
	}

	// Fetch the existing reading list from the database
	readingList, err := a.readingListModel.Get(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			a.notFoundResponse(w, r)
		default:
			a.serverErrorResponse(w, r, err)
		}
		return
	}

	// Parse the JSON request body into an input struct
	var input struct {
		Name        *string `json:"name"`
		Description *string `json:"description"`
	}

	err = a.readJSON(w, r, &input)
	if err != nil {
		a.badRequestResponse(w, r, err)
		return
	}

	// Update the fields in the reading list based on the input
	if input.Name != nil {
		readingList.Name = *input.Name
	}
	if input.Description != nil {
		readingList.Description = *input.Description
	}
	// if input.Books != nil {
	// 	readingList.Books = *input.Books
	// }
	// if input.Status != nil {
	// 	readingList.Status = *input.Status
	// }

	// Validate the updated reading list
	v := validator.New()
	data.ValidateReadingList(v, readingList)
	if !v.Valid() {
		a.failedValidationResponse(w, r, v.Errors)
		return
	}

	// Save the updated reading list to the database
	err = a.readingListModel.Update(readingList)
	if err != nil {
		a.serverErrorResponse(w, r, err)
		return
	}

	// Send the updated reading list in the response
	data := envelope{"reading_list": readingList}
	err = a.writeJSON(w, http.StatusOK, data, nil)
	if err != nil {
		a.serverErrorResponse(w, r, err)
	}
}

func (a *applicationDependencies) deleteReadingListHandler(w http.ResponseWriter, r *http.Request) {
	// Extract the reading list ID from the URL.
	id, err := a.readIDParam(r)
	if err != nil {
		a.notFoundResponse(w, r)
		return
	}

	// Delete the reading list from the database.
	err = a.readingListModel.Delete(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			a.notFoundResponse(w, r)
		default:
			a.serverErrorResponse(w, r, err)
		}
		return
	}

	// Respond with a success message.
	response := envelope{"message": "reading list successfully deleted"}
	err = a.writeJSON(w, http.StatusOK, response, nil)
	if err != nil {
		a.serverErrorResponse(w, r, err)
	}
}

func (a *applicationDependencies) addBookToReadingListHandler(w http.ResponseWriter, r *http.Request) {
	// Get the reading list ID from the URL parameters
	readingListID, err := a.readIDParam(r)
	if err != nil {
		a.notFoundResponse(w, r)
		return
	}

	// Decode the request body to get the book ID
	var input struct {
		BookID int    `json:"book_id"`
		Status string `json:"status"`
	}
	err = a.readJSON(w, r, &input)
	if err != nil {
		a.badRequestResponse(w, r, err)
		return
	}

	err = a.bookModel.BookExists(input.BookID)
	if err != nil {
		a.notFoundResponse(w, r)
		return
	}

	v := validator.New()
	data.ValidateBookInList(v, input.Status)
	if !v.Valid() {
		a.failedValidationResponse(w, r, v.Errors)
		return
	}

	err = a.readingListModel.ReadingListExist(readingListID)
	if err != nil {
		a.notFoundResponse(w, r)
		return
	}

	bookInList := &data.BookINlist{
		ListNameID: readingListID,
		BookID:     input.BookID,
		Status:     input.Status,
	}

	// Add the book to the reading list
	err = a.readingListModel.AddBook(bookInList)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			a.notFoundResponse(w, r)
		default:
			a.serverErrorResponse(w, r, err)
		}
		return
	}

	// Send a success response
	envelope := envelope{"message": "book successfully added to the reading list"}
	
	err = a.writeJSON(w, http.StatusOK, envelope, nil)
	if err != nil {
		a.serverErrorResponse(w, r, err)
	}
}

func (a *applicationDependencies) removeBookFromReadingListHandler(w http.ResponseWriter, r *http.Request) {
	// Get the reading list ID from the URL parameters
	readingListID, err := a.readIDParam(r)
	if err != nil {
		a.notFoundResponse(w, r)
		return
	}

	// Decode the request body to get the book ID
	var input struct {
		BookID int `json:"book_id"`
	}
	err = a.readJSON(w, r, &input)
	if err != nil {
		a.badRequestResponse(w, r, err)
		return
	}

	// Remove the book from the reading list
	err = a.readingListModel.RemoveBook(readingListID, input.BookID)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			a.notFoundResponse(w, r)
		default:
			a.serverErrorResponse(w, r, err)
		}
		return
	}

	// Send a success response
	envelope := envelope{"message": "book successfully removed from the reading list"}
	err = a.writeJSON(w, http.StatusOK, envelope, nil)
	if err != nil {
		a.serverErrorResponse(w, r, err)
	}
}
