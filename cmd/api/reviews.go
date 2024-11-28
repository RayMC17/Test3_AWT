package main

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/RayMC17/bookclub-api/internal/data"
	"github.com/RayMC17/bookclub-api/internal/validator"
)

// createReviewHandler handles the creation of a new review for a specific book.
func (a *applicationDependencies) createReviewHandler(w http.ResponseWriter, r *http.Request) {
	// Parse book ID from the URL
	bookid, err := a.readIDParam(r)
	if err != nil {
		a.notFoundResponse(w, r)
		return
	}

	// Define a structure to hold the expected data from the request body
	var input struct {
		AuthorID int    `json:"user_id"`
		Content  string `json:"review_text"`
		Rating   int    `json:"rating"`
	}

	// Parse JSON request body
	err = a.readJSON(w, r, &input)
	if err != nil {
		a.badRequestResponse(w, r, err)
		return
	}

	// Create a Review instance with the parsed data
	review := &data.Review{
		BookID:   int64(bookid),
		AuthorID: input.AuthorID,
		Content:  input.Content,
		Rating:   input.Rating,
	}

	// Validate the review data
	v := validator.New()
	data.ValidateReview(v, review)
	if !v.Valid() {
		a.failedValidationResponse(w, r, v.Errors)
		return
	}

	err = a.bookModel.BookExists(int(review.BookID))
	if err != nil {
		a.notFoundResponse(w, r)
		return
	}

	err = a.userModel.UserExists(input.AuthorID)
	if err != nil {
		a.notFoundResponse(w, r)
		return
	}

	// Insert the new review into the database
	err = a.reviewModel.Insert(review)
	if err != nil {
		a.serverErrorResponse(w, r, err)
		return
	}

	// Send a response with the created review
	headers := make(http.Header)
	headers.Set("Location", fmt.Sprintf("/api/v1/books/%d/reviews/%d", review.BookID, review.ID))
	err = a.writeJSON(w, http.StatusCreated, envelope{"review": review}, headers)
	if err != nil {
		a.serverErrorResponse(w, r, err)
	}
}

func (a *applicationDependencies) updateReviewHandler(w http.ResponseWriter, r *http.Request) {
	// Parse the review ID from the URL
	revID, err := a.readIDParam(r)
	if err != nil || revID < 1 {
		a.notFoundResponse(w, r)
		return
	}
	id64 := int64(revID)
	// Fetch the existing review from the database
	review, err := a.reviewModel.Get(id64)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrNoRecord):
			a.notFoundResponse(w, r)
		default:
			a.serverErrorResponse(w, r, err)
		}
		return
	}

	// Define a struct for holding the updated data
	var input struct {
		Content *string `json:"review_text"`
		Rating  *int    `json:"rating"`
	}

	// Parse the input from the request body
	err = a.readJSON(w, r, &input)
	if err != nil {
		a.badRequestResponse(w, r, err)
		return
	}

	// Update the review fields if new data is provided
	if input.Content != nil {
		review.Content = *input.Content
	}
	if input.Rating != nil {
		review.Rating = *input.Rating
	}

	// Validate the updated review
	v := validator.New()
	data.ValidateReview(v, review)
	if !v.Valid() {
		a.failedValidationResponse(w, r, v.Errors)
		return
	}

	// Save the updated review
	err = a.reviewModel.Update(review)
	if err != nil {
		a.serverErrorResponse(w, r, err)
		return
	}

	// Send the updated review in the response
	response := envelope{"review": review}
	err = a.writeJSON(w, http.StatusOK, response, nil)
	if err != nil {
		a.serverErrorResponse(w, r, err)
	}
}

func (a *applicationDependencies) deleteReviewHandler(w http.ResponseWriter, r *http.Request) {
	// Parse the review ID from the URL and convert it to int64
	revID, err := a.readIDParam(r)
	if err != nil || revID < 1 {
		a.notFoundResponse(w, r)
		return
	}

	// Convert the id to int64 if it's not already
	id64 := int64(revID)

	// Delete the review
	err = a.reviewModel.Delete(id64)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			a.notFoundResponse(w, r)
		default:
			a.serverErrorResponse(w, r, err)
		}
		return
	}

	//display the message
	data := envelope{
		"messaeg": "review deleted sucessfully",
	}

	err = a.writeJSON(w, http.StatusOK, data, nil)
	if err != nil {
		a.serverErrorResponse(w, r, err)
	}
}

func (a *applicationDependencies) listReviewsHandler(w http.ResponseWriter, r *http.Request) {

	bookID, err := a.readIDParam(r)
	if err != nil || bookID < 1 {
		a.notFoundResponse(w, r)
		return
	}

	err = a.bookModel.BookExists(bookID)
	if err != nil {
		a.notFoundResponse(w, r)
		return
	}

	reviews, err := a.reviewModel.GetAll(int64(bookID))
	if err != nil {
		a.serverErrorResponse(w, r, err)
		return
	}

	response := envelope{
		"reviews": reviews,
	}
	err = a.writeJSON(w, http.StatusOK, response, nil)
	if err != nil {
		a.serverErrorResponse(w, r, err)
	}
}
