package helper

import (
	"unicode"
)

func ConvertBodyToSnakeCase(test map[string]interface{}) map[string]interface{} {
	returnMap := make(map[string]interface{})
	for i, v := range test {
		switch v.(type) {
		case map[string]interface{}:
			returnMap[convertStringToSnake(i)] = ConvertBodyToSnakeCase(v.(map[string]interface{}))
		case []interface{}:
			// fmt.Println("i : ", i, " v : ", v)
			returnMap[convertStringToSnake(i)] = handleSliceOfInterface(v.([]interface{}))
		default:
			// Convert the name to snake case and send it here
			returnMap[convertStringToSnake(i)] = v

		}
	}
	return returnMap

}

func handleSliceOfInterface(test []interface{}) []interface{} {
	var returnInterface []interface{}
	for _, v := range test {
		switch v.(type) {
		case map[string]interface{}:
			returnInterface = append(returnInterface, ConvertBodyToSnakeCase(v.(map[string]interface{})))
		default:
			// if it is not a type of map[string]interface{}. The entire slice does not require any type of manipulation
			return test
		}
	}
	return returnInterface
}

func convertStringToSnake(in string) string {
	runes := []rune(in)
	length := len(runes)

	var out []rune
	for i := 0; i < length; i++ {
		if i > 0 && unicode.IsUpper(runes[i]) && ((i+1 < length && unicode.IsLower(runes[i+1])) || unicode.IsLower(runes[i-1])) {
			out = append(out, '_')
		}
		out = append(out, unicode.ToLower(runes[i]))
	}

	return string(out)
}
