package ffvm

import (
	"fmt"
	"reflect"
	"strings"
)

type (
	FuncCategory string

	LenInterface interface {
		Len() int
	}

	ValidatorIssue struct {
		Issue string `json:"issue,omitempty"`
		Field string `json:"field"`
		Value any    `json:"value"`
	}

	// Parser parse an actor syntax:
	// it has two parts separated by `,`: 1) manipulator(mapper, modifier), 2) validator
	Parser struct {
		// get some values and return single value which can be
		// anything, if it is bool the func will be considered as a validator
		mappers map[string]func(args ...string) func(any) any

		// validators are functions that receive the value and
		// an arbitrary count of arguments, these args all
		// are just strings that must be parsed in the function itself
		// return a single ValidatorIssue struct
		// if no error has been detected an empty ValidatorIssue should be returned
		validators map[string]func(args ...string) func(any) ValidatorIssue

		// each type has a list of actors
		// in the same order of reflect.Value.Field
		// each actor is a function that accept a value
		// then return the new value and the validation result
		actors map[reflect.Type][]func(self any) (any, []ValidatorIssue)
	}
)

const (
	Mapper    = FuncCategory("mapper")
	Validator = FuncCategory("validator")
)

var (
	NoIssue = ValidatorIssue{}

	HoldOldValue struct{}
)

func (p Parser) SetMapperGenerator(generatorName string, mapper func(args ...string) func(any) any) {
	if mapper == nil {
		return
	}

	p.mappers[generatorName] = mapper
}

func (p Parser) SetValidatorGenerator(generatorName string, validator func(args ...string) func(any) ValidatorIssue) {
	if validator == nil {
		return
	}

	p.validators[generatorName] = validator
}

func (p Parser) GenerateMapperFunc(generatorName string, args ...string) func(any) any {
	f, exist := p.mappers[generatorName]
	if !exist {
		panic("mapper with name='" + generatorName + "' does not exist")
	}

	return f(args...)
}

func (p Parser) GenerateValidatorFunc(generatorName string, args ...string) func(any) ValidatorIssue {
	f, exist := p.validators[generatorName]
	if !exist {
		panic("validator with name='" + generatorName + "' does not exist")
	}

	return f(args...)
}

// parseActors parse all exported actors and save them in a map
func (p Parser) parseActors(instance any) {
	val := reflect.ValueOf(instance)
	if val.Kind() != reflect.Struct {
		panic("instance must be struct but was " + val.Kind().String())
	}

	t := val.Type()
	if _, exist := p.actors[t]; exist {
		return
	}

	actors := make([]func(any) (any, []ValidatorIssue), t.NumField())
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldAddr := nameOfField(field)

		var mapperFunc func(any) any
		var validatorFunc func(any) []ValidatorIssue

		tag, exist := field.Tag.Lookup("ffvm")
		if exist && tag != "" {
			if !strings.Contains(tag, ",") {
				tag = "," + tag
			}

			tokens := strings.Split(tag, ",")
			mappers, validators := strings.Split(tokens[0], ";"), strings.Split(tokens[1], ";")

			mapperFunc = p.createMapperFuncOfField(mappers)
			validatorFunc = p.createValidatorFuncOfField(validators)
		}

		actors[i] = func(self any) (any, []ValidatorIssue) {
			newVal, issues := p.act(self, fieldAddr)

			if mapperFunc != nil {
				newVal = mapperFunc(newVal)
			}

			if validatorFunc != nil {
				issues = append(issues, validatorFunc(newVal)...)
			}

			return newVal, issues
		}
	}

	var foundNotNilActor bool
	for _, actor := range actors {
		if actor != nil {
			foundNotNilActor = true
			break
		}
	}

	if !foundNotNilActor {
		p.actors = nil
		return
	}

	p.actors[t] = actors
}

func (p Parser) createValidatorFuncOfField(validators []string) func(any) []ValidatorIssue {
	generatedValidators := make([]func(any) ValidatorIssue, len(validators))
	for i, validator := range validators {
		validator, args := tokenizeFunction(validator)
		left := p.GenerateValidatorFunc(validator, args...)
		generatedValidators[i] = func(self any) ValidatorIssue {
			return left(self)
		}
	}

	return func(self any) []ValidatorIssue {
		var issues []ValidatorIssue
		for _, validator := range generatedValidators {
			if issue := validator(self); issue != NoIssue {
				issue.Value = self
				issues = append(issues, issue)
			}
		}

		return issues
	}
}

func (p Parser) createMapperFuncOfField(mappers []string) func(any) any {
	var mapperFunc func(any) any
	for _, mapper := range mappers {
		mapper, args := tokenizeFunction(mapper)
		innerFunc := p.GenerateMapperFunc(mapper, args...)
		if mapperFunc == nil {
			mapperFunc = innerFunc
		} else {
			outer := mapperFunc
			mapperFunc = func(self any) any { return outer(innerFunc(self)) }
		}
	}

	return mapperFunc
}

// act firstly do the manipulation, then validation
// instancePtr is a pointer to an instance of the struct
func (p Parser) act(in any, parent string) (any, []ValidatorIssue) {
	val := reflect.ValueOf(in)
	switch val.Kind() {
	case reflect.Struct: // continue function
	case reflect.Pointer: // continue function
		val = val.Elem()
	case reflect.Slice, reflect.Array:
		var allIssues []ValidatorIssue
		for i := 0; i < val.Len(); i++ {
			addr := fmt.Sprintf("%s[%d]", parent, i)
			newVal, issues := p.act(val.Index(i).Interface(), addr)
			val.Index(i).Set(reflect.ValueOf(newVal))
			allIssues = append(allIssues, issues...)
		}

		return in, allIssues
	case reflect.Map:
		var allIssues []ValidatorIssue
		iter := val.MapRange()
		for iter.Next() {
			addr := fmt.Sprint(parent, ".", iter.Key().Interface())
			newVal, issues := p.act(iter.Value().Interface(), addr)
			val.SetMapIndex(iter.Key(), reflect.ValueOf(newVal))
			allIssues = append(allIssues, issues...)
		}

		return in, allIssues
	default:
		return in, nil
	}

	t := val.Type()
	actors, exist := p.actors[t]
	if !exist {
		p.parseActors(val.Interface())
		actors = p.actors[t]
	}

	if actors == nil {
		return in, nil
	}

	var issues []ValidatorIssue
	for i, actor := range actors {
		if actor == nil {
			continue
		}

		field := val.Field(i)
		mapped, fieldIssues := actor(field.Interface())
		mappedVal := reflect.ValueOf(mapped)
		if mapped != HoldOldValue && field.CanSet() && field.CanConvert(mappedVal.Type()) {
			field.Set(mappedVal)
		}
		addr := nameOfField(val.Type().Field(i))
		if parent != "" {
			addr = parent + "." + addr
		}

		for i := range fieldIssues {
			if fieldIssues[i].Field != "" {
				fieldIssues[i].Field = fmt.Sprint(addr, ".", fieldIssues[i].Field)
			} else {
				fieldIssues[i].Field = addr
			}
		}

		issues = append(issues, fieldIssues...)
	}

	return in, issues
}

func tokenizeFunction(f string) (string, []string) {
	index := strings.Index(f, "=")
	if index <= 0 {
		return f, nil
	}

	return f[:index], strings.Split(f[index+1:], "&")
}
