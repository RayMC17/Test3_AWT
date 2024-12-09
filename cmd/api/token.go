package main

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/RayMC17/bookclub-api/internal/data"
	"github.com/RayMC17/bookclub-api/internal/validator"
)

func (a *applicationDependencies) authenticateTokenHandler(w http.ResponseWriter, r *http.Request) {
	var incomingData struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	err := a.readJSON(w, r, &incomingData)
	if err != nil {
		a.badRequestResponse(w, r, err)
		return
	}

	v := validator.New()
	data.ValidateEmail(v, incomingData.Email)
	data.ValidatePassword(v, incomingData.Password)

	if !v.Valid() {
		a.failedValidationResponse(w, r, v.Errors)
		return
	}

	user, err := a.userModel.GetByEmail(incomingData.Email)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			a.invalidCredentialResponse(w, r)
		default:
			a.serverErrorResponse(w, r, err)
		}
		return
	}

	matches, err := user.Password.Matches(incomingData.Password)
	if err != nil {
		a.serverErrorResponse(w, r, err)
		return
	}

	if !matches {
		a.invalidCredentialResponse(w, r)
		return
	}

	token, err := a.tokenModel.New(int64(user.ID), 24*time.Hour, data.ScopeAuthentication)
	if err != nil {
		a.serverErrorResponse(w, r, err)
		return
	}

	data := envelope{
		"authentication_token": token,
	}

	err = a.writeJSON(w, http.StatusCreated, data, nil)
	if err != nil {
		a.serverErrorResponse(w, r, err)
		return
	}

}

//Step 2: Implement Password reset (Generate Password Reset Token and )

func (a *applicationDependencies) createPasswordResetTokenHandler(w http.ResponseWriter, r *http.Request) {
	// Get the passed-in email address from the request body
	var incomingData struct {
		Email string `json:"email"`
	}
	err := a.readJSON(w, r, &incomingData)
	if err != nil {
		a.badRequestResponse(w, r, err)
		return
	}

	// Fetch user by email address
	user, err := a.userModel.GetByEmail(incomingData.Email)
	if err != nil {
		// If no user is found, assume it's not registered; return 404 to avoid leaking information
		a.notFoundResponse(w, r)
		return
	}

	// Generate a password reset token
	token, err := a.tokenModel.New(int64(user.ID), 1*time.Hour, data.ScopePasswordReset)

	if err != nil {
		a.serverErrorResponse(w, r, err)
		return
	}

	data := envelope{
		"message": "An email will be sent with password reset instructions",
	}
	a.background(func() {
		emailData := map[string]any{
			"resetToken": token.PlainText, // Send the plaintext token in the email
			"userID":     user.ID,
		}

		err = a.mailer.Send(user.Email, "reset_password.tmpl", emailData)
		if err != nil {
			a.logger.Error("failed to send password reset email: " + err.Error())
		}
	})

	// Respond with a success message (don't send the token in the response)

	err = a.writeJSON(w, http.StatusOK, data, nil)
	if err != nil {
		a.serverErrorResponse(w, r, err)
		return
	}
}

// Update Password Using token
func (a *applicationDependencies) resetPasswordHandler(w http.ResponseWriter, r *http.Request) {
	// Read the token and new password from the request body
	var input struct {
		Password string `json:"password"`
		Token    string `json:"token"`
	}
	err := a.readJSON(w, r, &input)
	if err != nil {
		a.badRequestResponse(w, r, err)
		return
	}

	// Validate that the new password is not empty
	if input.Password == "" {
		a.badRequestResponse(w, r, fmt.Errorf("new password must be provided"))
		return
	}

	// Validate the token
	v := validator.New()
	data.ValidatetokenPlaintext(v, input.Token)
	if !v.Valid() {
		a.failedValidationResponse(w, r, v.Errors)
		return
	}

	// Check if the token is valid and belongs to the user
	user, err := a.userModel.GetForToken(data.ScopePasswordReset, input.Token)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			v.AddError("token", "invalid or expired activation token")
			a.failedValidationResponse(w, r, v.Errors)
		default:
			a.serverErrorResponse(w, r, err)
		}
		return
	}

	// Fetch the user associated with the token
	user, err = a.userModel.GetByID(int64(user.ID))
	if err != nil {
		a.notFoundResponse(w, r)
		return
	}

	err = user.Password.Set(input.Password)
	if err != nil {
		a.serverErrorResponse(w, r, err)
		return
	}

	data.ValidateUser(v, user)
	if !v.Valid() {
		a.failedValidationResponse(w, r, v.Errors)
		return
	}

	err = a.userModel.Update(user)
	if err != nil {
		a.serverErrorResponse(w, r, err)
		return
	}

	// Delete all existing password reset tokens for the user
	err = a.tokenModel.DeleteAllForUser(data.ScopePasswordReset, int64(user.ID))
	if err != nil {
		a.serverErrorResponse(w, r, err)
		return
	}

	// Respond with a success message
	data := envelope{
		"message": "your password was successfully reset",
	}
	err = a.writeJSON(w, http.StatusOK, data, nil)
	if err != nil {
		a.serverErrorResponse(w, r, err)
		return
	}
}
