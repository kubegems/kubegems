package request

import (
	"encoding/json"
	"encoding/xml"
	"net/http"
	"strconv"
)

// nolint: forcetypeassert,gomnd,ifshort
func Query[T any](r *http.Request, key string, defaultValue T) T {
	val := r.URL.Query().Get(key)
	if val == "" {
		return defaultValue
	}
	switch any(defaultValue).(type) {
	case string:
		return any(val).(T)
	case int:
		intval, _ := strconv.Atoi(val)
		return any(intval).(T)
	case bool:
		b, _ := strconv.ParseBool(val)
		return any(b).(T)
	case int64:
		intval, _ := strconv.ParseInt(val, 10, 64)
		return any(intval).(T)
	default:
		return defaultValue
	}
}

func Body(r *http.Request, into any) error {
	switch r.Header.Get("Content-Type") {
	case "application/json", "":
		return json.NewDecoder(r.Body).Decode(into)
	case "application/xml":
		return xml.NewDecoder(r.Body).Decode(into)
	}
	return nil
}
