package encode

import (
	"encoding/json"

	"github.com/shopspring/decimal"
)

func MustJSONMarshal(v interface{}) []byte {
	encoded, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return encoded
}

func DecimalToFloat(d decimal.Decimal) (f float64) {
	f, _ = d.Float64()
	return
}
