package ffvm

import (
	"fmt"
	"reflect"
	"strings"
)

func CreateDefaultIssue(issue ...any) ValidatorIssue {
	return ValidatorIssue{Issue: fmt.Sprint(issue...)}
}

func getLen(self any) int {
	val := reflect.ValueOf(self)
	switch val.Kind() {
	case reflect.String, reflect.Map, reflect.Array, reflect.Slice, reflect.Chan:
		return val.Len()
	case reflect.Pointer:
		if val.Elem().Kind() == reflect.Array {
			return val.Len()
		}
	}

	if lenInter, ok := self.(LenInterface); ok {
		return lenInter.Len()
	}

	return -1
}

func argCountPanic(args []string, minLen, maxLen int, funcName, category FuncCategory) {
	if len(args) > maxLen {
		panic(fmt.Sprint(category, " name=", funcName, " does not accept more than ",
			maxLen, " argument(s) but has ", len(args), " argument(s)"))
	}

	if len(args) < minLen {
		panic(fmt.Sprint(category, " name=", funcName, " does not accept less than ",
			minLen, " argument(s) but has ", len(args), " argument(s)"))
	}
}

func getAsFloat64(self any) (float64, bool) {
	switch v := reflect.ValueOf(self); self.(type) {
	case int, int8, int16, int32, int64:
		return float64(v.Int()), true
	case uint, uint8, uint16, uint32, uint64:
		return float64(v.Uint()), true
	case float32, float64:
		return v.Float(), true
	default:
		return 0, false
	}
}

func isEmpty(object any) bool {

	// get nil case out of the way
	if object == nil {
		return true
	}

	objValue := reflect.ValueOf(object)

	switch objValue.Kind() {
	// collection types are empty when they have no element
	case reflect.Array, reflect.Chan, reflect.Map, reflect.Slice:
		return objValue.Len() == 0
		// pointers are empty if nil or if the value they point to is empty
	case reflect.Ptr:
		if objValue.IsNil() {
			return true
		}
		deref := objValue.Elem().Interface()
		return isEmpty(deref)
		// for all other types, compare against the zero value
	default:
		zero := reflect.Zero(objValue.Type())
		return reflect.DeepEqual(object, zero.Interface())
	}
}

func nameOfField(field reflect.StructField) string {
	jsonTag, ok := field.Tag.Lookup("json")
	if !ok {
		return field.Name
	}

	if strings.Contains(jsonTag, ",") {
		jsonTag = jsonTag[:strings.Index(jsonTag, ",")]
	}

	return jsonTag
}
