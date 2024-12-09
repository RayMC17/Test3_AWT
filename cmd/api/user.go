package main

import (
	"errors"

	//"fmt"
	"net/http"
	// "strconv"
	"time"

	"github.com/RayMC17/bookclub-api/internal/data"
	"github.com/RayMC17/bookclub-api/internal/validator"
)

func (a *applicationDependencies) makeUserProfileHandler(w http.ResponseWriter, r *http.Request) {
	var incomingData struct {
		UserName string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	err := a.readJSON(w, r, &incomingData)
	if err != nil {
		a.badRequestResponse(w, r, err)
		return
	}

	user := &data.User{
		Username:  incomingData.UserName,
		Email:     incomingData.Email,
		Activated: false,
	}

	err = user.Password.Set(incomingData.Password)
	if err != nil {
		a.serverErrorResponse(w, r, err)
		return
	}

	v := validator.New()
	data.ValidateUser(v, user)
	if !v.Valid() {
		a.failedValidationResponse(w, r, v.Errors)
		return
	}

	err = a.userModel.Insert(user)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrDuplicateEmail):
			v.AddError("email", "email is already in use")
			a.failedValidationResponse(w, r, v.Errors)
		default:
			a.serverErrorResponse(w, r, err)
		}
		return
	}

	token, err := a.tokenModel.New(int64(user.ID), 3*24*time.Hour, data.ScopeActivation)
	if err != nil {
		a.serverErrorResponse(w, r, err)
		return
	}

	data := envelope{
		"user": user,
	}

	a.background(func() {
		data := map[string]any{
			"activationToken": token.PlainText,
			"userID":          user.ID,
		}
		err = a.mailer.Send(user.Email, "mail_tmpl.tmpl", data)
		if err != nil {
			a.logger.Error(err.Error())
		}

	})

	//status code 201 resource created
	err = a.writeJSON(w, http.StatusCreated, data, nil)
	if err != nil {
		a.serverErrorResponse(w, r, err)
		return
	}
}

func (a *applicationDependencies) getUserProfileHandler(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from the URL parameters
	id, err := a.readIDParam(r)
	if err != nil {
		a.notFoundResponse(w, r)
		return
	}

	// Get the user profile from the database using the user model
	profile, err := a.userModel.Get(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrNoRecord):
			a.notFoundResponse(w, r)
		default:
			a.serverErrorResponse(w, r, err)
		}
		return
	}

	// Respond with the user profile data in JSON format
	err = a.writeJSON(w, http.StatusOK, envelope{"user_profile": profile}, nil)
	if err != nil {
		a.serverErrorResponse(w, r, err)
	}
}

func (a *applicationDependencies) getUserReadingListsHandler(w http.ResponseWriter, r *http.Request) {
	// Get the user ID from the URL parameters
	userID, err := a.readIDParam(r)
	if err != nil {
		a.notFoundResponse(w, r)
		return
	}

	// Get the reading lists associated with the user from the model
	readingLists, err := a.readingListModel.GetAllByUser(int64(userID))
	if err != nil {
		a.serverErrorResponse(w, r, err)
		return
	}

	// Respond with the reading lists and metadata in JSON format
	response := envelope{
		"reading_lists": readingLists,
	}
	err = a.writeJSON(w, http.StatusOK, response, nil)
	if err != nil {
		a.serverErrorResponse(w, r, err)
	}
}

func (a *applicationDependencies) getUserReviewsHandler(w http.ResponseWriter, r *http.Request) {
	// Get the user ID from the URL parameters
	id, err := a.readIDParam(r)
	if err != nil {
		a.notFoundResponse(w, r)
		return
	}

	// Get the reviews associated with the user from the model
	reviews, err := a.reviewModel.GetAllByUser(int64(id))
	if err != nil {
		a.serverErrorResponse(w, r, err)
		return
	}

	// Respond with the reviews and metadata in JSON format
	response := envelope{
		"reviews": reviews,
	}
	err = a.writeJSON(w, http.StatusOK, response, nil)
	if err != nil {
		a.serverErrorResponse(w, r, err)
	}
}

func (a *applicationDependencies) activateUserhandler(w http.ResponseWriter, r *http.Request) {
	var incomingData struct {
		TokenText string `json:"token"`
	}

	err := a.readJSON(w, r, &incomingData)

	if err != nil {
		a.badRequestResponse(w, r, err)
		return
	}

	v := validator.New()
	data.ValidatetokenPlaintext(v, incomingData.TokenText)
	if !v.Valid() {
		a.failedValidationResponse(w, r, v.Errors)
		return
	}

	usr, err := a.userModel.GetForToken(data.ScopeActivation, incomingData.TokenText)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			v.AddError("token", "invalid or expired token")
			a.failedValidationResponse(w, r, v.Errors)
		default:
			a.serverErrorResponse(w, r, err)
		}
		return
	}

	usr.Activated = true
	err = a.userModel.Update(usr)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrEditConfilct):
			a.editConflictResponse(w, r)
		default:
			a.serverErrorResponse(w, r, err)
		}
		return
	}

	err = a.tokenModel.DeleteAllForUser(data.ScopeActivation, int64(usr.ID))
	if err != nil {
		a.serverErrorResponse(w, r, err)
		return
	}

	//send response
	data := envelope{
		"user": usr,
	}
	err = a.writeJSON(w, http.StatusOK, data, nil)
	if err != nil {
		a.serverErrorResponse(w, r, err)
		return
	}
}
