package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	// "github.com/RayMC17/bookclub-api/internal/data"
	"github.com/RayMC17/bookclub-api/internal/validator"
	"github.com/julienschmidt/httprouter"
)

type envelope map[string]any

// writeJSON writes a response in JSON format.
func (a *applicationDependencies) writeJSON(w http.ResponseWriter, status int, data envelope, headers http.Header) error {
	js, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		return err
	}
	js = append(js, '\n')

	for key, value := range headers {
		w.Header()[key] = value
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, err = w.Write(js)
	return err
}

// readJSON reads JSON data from the request body and decodes it into the destination struct.
func (a *applicationDependencies) readJSON(w http.ResponseWriter, r *http.Request, dst interface{}) error {
	maxBytes := 1_048_576 // 1 MB
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes))

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	err := dec.Decode(dst)
	if err != nil {
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError
		var maxBytesError *http.MaxBytesError

		switch {
		case errors.As(err, &syntaxError):
			return fmt.Errorf("body contains badly-formed JSON (at character %d)", syntaxError.Offset)
		case errors.Is(err, io.ErrUnexpectedEOF):
			return errors.New("body contains badly-formed JSON")
		case errors.As(err, &unmarshalTypeError):
			return fmt.Errorf("body contains incorrect JSON type for field %q", unmarshalTypeError.Field)
		case errors.Is(err, io.EOF):
			return errors.New("body must not be empty")
		case strings.HasPrefix(err.Error(), "json: unknown field "):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
			return fmt.Errorf("body contains unknown key %s", fieldName)
		case errors.As(err, &maxBytesError):
			return fmt.Errorf("body must not be larger than %d bytes", maxBytesError.Limit)
		default:
			return err
		}
	}

	err = dec.Decode(&struct{}{})
	if err != io.EOF {
		return errors.New("body must contain only a single JSON value")
	}

	return nil
}

// getSingleQueryParameter returns a single query parameter value or a default value if not present.
func (a *applicationDependencies) getSingleQueryParameter(queryParameters url.Values, key string, defaultValue string) string {
	result := queryParameters.Get(key)
	if result == "" {
		return defaultValue
	}
	return result
}

// getSingleIntegerParameter returns an integer query parameter value or a default value if not present.
func (a *applicationDependencies) getSingleIntegerParameter(queryParameters url.Values, key string, defaultValue int, v *validator.Validator) int {
	result := queryParameters.Get(key)
	if result == "" {
		return defaultValue
	}

	intValue, err := strconv.Atoi(result)
	if err != nil {
		v.AddError(key, "must be an integer value")
		return defaultValue
	}

	return intValue
}

// readIDParam extracts an integer ID parameter from the URL.
func (a *applicationDependencies) readIDParam(r *http.Request) (int, error) {
	params := httprouter.ParamsFromContext(r.Context())
	id, err := strconv.Atoi(params.ByName("id"))
	if err != nil || id < 1 {
		return 0, errors.New("invalid ID parameter")
	}
	return id, nil
}

func (a *applicationDependencies) background(fn func()) {
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		defer func() {
			err := recover()
			if err != nil {
				a.logger.Error(fmt.Sprintf("%v", err))
			}
		}()
		fn() //running the function that was passed to run as parameter
	}()
}
