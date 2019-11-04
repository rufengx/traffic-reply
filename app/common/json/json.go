package json

import (
	"bytes"
	"encoding/json"
)

func JsonEncode(obj interface{}, prettyPrint bool, printNull bool, safePrint bool) string {
	b, err := jsonEncode(obj, prettyPrint, printNull)
	if err != nil {
		panic(err.Error())
	}
	if safePrint {
		b = bytes.Replace(b, []byte("\\u003c"), []byte("<"), -1)
		b = bytes.Replace(b, []byte("\\u003e"), []byte(">"), -1)
		b = bytes.Replace(b, []byte("\\u0026"), []byte("&"), -1)
	}
	return string(b)
}

func jsonEncode(obj interface{}, prettyPrint bool, printNull bool) ([]byte, error) {
	if obj == nil && !printNull {
		obj = struct{}{}
	}

	if prettyPrint {
		return json.MarshalIndent(obj, "", "    ")
	} else {
		return json.Marshal(obj)
	}
}
