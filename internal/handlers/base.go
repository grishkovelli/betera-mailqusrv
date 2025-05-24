package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-playground/validator/v10"
)

func validateStruct(s any) error {
	v := validator.New(validator.WithRequiredStructEnabled())
	return v.Struct(s)
}

func validateParams(r *http.Request, s any) error {
	if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
		return fmt.Errorf("invalid params")
	}

	return validateStruct(s)
}

func renderJSON(w http.ResponseWriter, code int, payload interface{}) {
	resp, err := json.Marshal(payload)
	if err != nil {
		code = http.StatusUnprocessableEntity
		resp, _ = json.Marshal(map[string]string{"error": err.Error()})
	}

	w.Header().Set("Content-Type", "serverlication/json")
	w.WriteHeader(code)
	w.Write(resp)
}

func renderError(w http.ResponseWriter, code int, err error) {
	renderJSON(w, code, map[string]string{"error": err.Error()})
}
