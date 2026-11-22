package httpx

import (
	"encoding/json"
	"math"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Float — число для JSON-ответа в записи Python json.dumps (repr):
// 55.0, а не 55; экспонента при 1e16 и больше или меньше 1e-4.
type Float float64

func (f Float) MarshalJSON() ([]byte, error) {
	v := float64(f)
	if math.IsInf(v, 0) || math.IsNaN(v) {
		return nil, &json.UnsupportedValueError{Str: strconv.FormatFloat(v, 'g', -1, 64)}
	}
	e := strconv.FormatFloat(v, 'e', -1, 64) // кратчайшая запись, как repr
	exp, _ := strconv.Atoi(e[strings.IndexByte(e, 'e')+1:])
	if v != 0 && (exp < -4 || exp >= 16) {
		return []byte(e), nil // Go и Python пишут одинаково: 1e+16, 1.5e-05
	}
	s := strconv.FormatFloat(v, 'f', -1, 64)
	if !strings.ContainsRune(s, '.') {
		s += ".0"
	}
	return []byte(s), nil
}

// Float — FloatField DRF: число или строка с числом, только конечные значения.
func (p *Payload) Float(key string, required bool) (float64, bool) {
	v, ok := p.lookup(key, required)
	if !ok {
		return 0, false
	}
	var raw string
	switch t := v.(type) {
	case json.Number:
		raw = t.String()
	case string:
		if utf8.RuneCountInString(t) > 1000 {
			p.Errors.Add(key, "Слишком длинное значение.")
			return 0, false
		}
		raw = strings.TrimSpace(t)
	default:
		p.Errors.Add(key, "Требуется численное значение.")
		return 0, false
	}
	f, err := strconv.ParseFloat(raw, 64)
	if err != nil || math.IsInf(f, 0) || math.IsNaN(f) || strings.HasPrefix(strings.ToLower(raw), "0x") {
		p.Errors.Add(key, "Требуется численное значение.")
		return 0, false
	}
	return f, true
}
