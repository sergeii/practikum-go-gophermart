package testutils

import "encoding/json"

func MustJSONMarshal(v interface{}) []byte {
	encoded, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return encoded
}
