package httpx

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const maxBody = 1 << 20

// Payload — тело запроса в JSON, form-urlencoded или multipart (как парсеры DRF).
type Payload struct {
	values map[string]any
	Errors FieldErrors
}

func DecodePayload(r *http.Request) (*Payload, error) {
	p := &Payload{values: map[string]any{}, Errors: FieldErrors{}}
	ct, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
	switch ct {
	case "application/json", "":
		raw, err := io.ReadAll(http.MaxBytesReader(nil, r.Body, maxBody))
		if err != nil {
			return nil, err
		}
		if len(bytes.TrimSpace(raw)) == 0 {
			break // пустое тело допустимо, например PATCH без полей
		}
		dec := json.NewDecoder(bytes.NewReader(raw))
		dec.UseNumber()
		var v any
		if err := dec.Decode(&v); err != nil {
			return nil, fmt.Errorf("JSON parse error - %v", err)
		}
		obj, ok := v.(map[string]any)
		if !ok {
			return nil, errors.New("Недопустимые данные. Ожидался словарь.")
		}
		p.values = obj
	case "multipart/form-data":
		if err := r.ParseMultipartForm(maxBody); err != nil {
			return nil, err
		}
		for k, v := range r.MultipartForm.Value {
			p.values[k] = v[0]
		}
	case "application/x-www-form-urlencoded":
		r.Body = http.MaxBytesReader(nil, r.Body, maxBody)
		if err := r.ParseForm(); err != nil {
			return nil, err
		}
		for k, v := range r.PostForm {
			p.values[k] = v[0]
		}
	default:
		return nil, fmt.Errorf("Неподдерживаемый тип данных «%s» в запросе.", ct)
	}
	if p.values == nil {
		p.values = map[string]any{}
	}
	return p, nil
}

// BadRequest пишет ошибку разбора тела.
func BadRequest(w http.ResponseWriter, err error) { Error(w, http.StatusBadRequest, err.Error()) }

func (p *Payload) Has(key string) bool { _, ok := p.values[key]; return ok }

// lookup возвращает значение; для обязательного отсутствующего поля пишет ошибку.
func (p *Payload) lookup(key string, required bool) (any, bool) {
	v, ok := p.values[key]
	if !ok {
		if required {
			p.Errors.Add(key, "Обязательное поле.")
		}
		return nil, false
	}
	if v == nil {
		p.Errors.Add(key, "Это поле не может быть null.")
		return nil, false
	}
	return v, true
}

func (p *Payload) String(key string, required, allowBlank bool, maxLen int) (string, bool) {
	v, ok := p.lookup(key, required)
	if !ok {
		return "", false
	}
	var s string
	switch t := v.(type) {
	case string:
		s = t
	case json.Number:
		s = t.String()
	case bool:
		s = strconv.FormatBool(t)
	default:
		p.Errors.Add(key, "Not a valid string.")
		return "", false
	}
	s = strings.TrimSpace(s)
	if s == "" && !allowBlank {
		p.Errors.Add(key, "Это поле не может быть пустым.")
		return "", false
	}
	if maxLen > 0 && utf8.RuneCountInString(s) > maxLen {
		p.Errors.Add(key, fmt.Sprintf("Убедитесь, что это значение содержит не более %d символов.", maxLen))
		return "", false
	}
	return s, true
}

func (p *Payload) Int(key string, required bool) (int, bool) {
	v, ok := p.lookup(key, required)
	if !ok {
		return 0, false
	}
	var raw string
	switch t := v.(type) {
	case json.Number:
		raw = t.String()
	case string:
		raw = strings.TrimSpace(t)
	default:
		p.Errors.Add(key, "Требуется целое число.")
		return 0, false
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		// DRF принимает "3.0", но не "3.5"
		if f, ferr := strconv.ParseFloat(raw, 64); ferr == nil && f == float64(int(f)) {
			return int(f), true
		}
		p.Errors.Add(key, "Требуется целое число.")
		return 0, false
	}
	return n, true
}

func (p *Payload) Bool(key string, required bool) (bool, bool) {
	v, ok := p.lookup(key, required)
	if !ok {
		return false, false
	}
	switch t := v.(type) {
	case bool:
		return t, true
	case json.Number:
		switch t.String() {
		case "1":
			return true, true
		case "0":
			return false, true
		}
	case string:
		switch strings.ToLower(strings.TrimSpace(t)) {
		case "true", "1", "yes", "y", "t", "on":
			return true, true
		case "false", "0", "no", "n", "f", "off":
			return false, true
		}
	}
	p.Errors.Add(key, "Требуется логическое значение.")
	return false, false
}

func (p *Payload) Date(key string, required bool) (time.Time, bool) {
	s, ok := p.String(key, required, false, 0)
	if !ok {
		return time.Time{}, false
	}
	d, err := ParseDate(s)
	if err != nil {
		p.Errors.Add(key, "Неправильный формат date. Используйте один из этих форматов: YYYY-MM-DD.")
		return time.Time{}, false
	}
	return d, true
}

func ParseDate(s string) (time.Time, error) { return time.Parse(time.DateOnly, s) }
