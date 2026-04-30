package handlers

import (
	"encoding/json"
	"net/http"
)

func ReadRequestBody(r *http.Request, data interface{}) error {
	decoder := json.NewDecoder(r.Body)
	// decoder.DisallowUnknownFields() Раскоментить, когда добавлю поддержку ссылки на конфу
	if err := decoder.Decode(data); err != nil {
		return err
	}
	return nil
}
