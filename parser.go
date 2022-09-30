package ffvm

import (
	"fmt"
	"reflect"
	"strings"
)

func NewParser() Parser {
	p := Parser{
		mappers:    map[string]func(in ...reflect.Value) reflect.Value{},
		validators: map[string]func(in ...reflect.Value) ValidatorIssue{},
		actors:     map[reflect.Type][]reflect.Value{},
	}

	p.SetMapperFunc("upper", func(str string) string {
		return strings.ToUpper(str)
	})

	p.SetMapperFunc("lower", func(str string) string {
		return strings.ToLower(str)
	})

	p.SetMapperFunc("len", func(v any) int { return reflect.ValueOf(v).Len() })

	p.SetValidatorFunc("not_nil", func(v any) ValidatorIssue {
		if v == nil {
			return ValidatorIssue{
				Issue: "expected not nil value",
			}
		}

		return ValidatorIssue{}
	})

	p.SetValidatorFunc("nil", func(v any) ValidatorIssue {
		if v != nil {
			return ValidatorIssue{
				Issue: "expected nil value but, " + fmt.Sprint(v),
			}
		}

		return ValidatorIssue{}
	})

	p.SetValidatorFunc("empty", func(v any) ValidatorIssue {
		if !reflect.DeepEqual(reflect.ValueOf(v), reflect.Zero(reflect.TypeOf(v))) {
			return ValidatorIssue{
				Issue: "expected empty value but, " + fmt.Sprint(v),
			}
		}

		return ValidatorIssue{}
	})

	p.SetMapperFunc("not_empty", func(v any) ValidatorIssue {
		if reflect.DeepEqual(reflect.ValueOf(v), reflect.Zero(reflect.TypeOf(v))) {
			return ValidatorIssue{
				Issue: "expected not empty value",
			}
		}

		return ValidatorIssue{}
	})

	p.SetValidatorFunc("upper", func(v string) ValidatorIssue {
		if v != strings.ToUpper(v) {
			return ValidatorIssue{
				Issue: "expected upper case",
			}
		}

		return ValidatorIssue{}
	})

	p.SetValidatorFunc("lower", func(v string) ValidatorIssue {
		if v != strings.ToLower(v) {
			return ValidatorIssue{
				Issue: "expected lower case",
			}
		}

		return ValidatorIssue{}
	})

	return p
}

type ValidatorIssue struct {
	Issue string
	Level string
}

// Parser parse an actor syntax:
// it has two parts separated by `,`: 1) manipulator(mapper, modifier), 2) validator
type Parser struct {
	// get some values and return single value which can be
	// anything, if it is bool the func will be considered as a validator
	mappers map[string]func(in ...reflect.Value) reflect.Value

	// validators are functions that receive the value and
	// return a single ValidatorIssue struct
	// if no error has been detected an empty ValidatorIssue should be returned
	validators map[string]func(in ...reflect.Value) ValidatorIssue

	// each type has a list of actors
	// in the same order of reflect.Value.Field
	// each actor is a function that accept a value
	// then return the new value and the validation result
	actors map[reflect.Type][]reflect.Value
}

func (p Parser) SetMapperFunc(funcName string, mapper any) {
	val := reflect.ValueOf(mapper)
	p.mappers[funcName] = func(in ...reflect.Value) reflect.Value {
		return val.Call(in)[0]
	}
}

func (p Parser) SetValidatorFunc(funcName string, validator any) {
	val := reflect.ValueOf(validator)
	p.validators[funcName] = func(in ...reflect.Value) ValidatorIssue {
		return val.Call(in)[0].Interface().(ValidatorIssue)
	}
}

// ParseActors parse all exported actors and save them in a map
func (p Parser) ParseActors(instance any) {
	val := reflect.ValueOf(instance)
	if val.Kind() != reflect.Struct {
		panic("instance must be struct")
	}

	t := val.Type()
	if _, exist := p.actors[t]; exist {
		return
	}

	var actors []reflect.Value
	for i := 0; i < t.NumField(); i++ {
		if tag, exist := t.Field(i).Tag.Lookup("ffvm"); exist {
			tokens := strings.Split(tag, ",")
			mappers, validators := strings.Split(tokens[0], ";"), strings.Split(tokens[1], ";")
			actors = append(actors, reflect.ValueOf(func(val reflect.Value) (issues []ValidatorIssue) {
				for _, mapper := range mappers {
					val.Set(p.mappers[mapper](val))
				}

				for _, validator := range validators {
					if validator == "" {
						continue
					}

					issue := p.validators[validator](val)
					if issue.Issue == "" {
						continue
					} else if issue.Level == "" {
						issue.Level = "UNKNOWN"
					}

					issues = append(issues, issue)
				}

				return issues
			}))
		}
	}

	p.actors[t] = actors
}

// Act firstly do the manipulation, then validation
// instancePtr is a pointer to an instance of the struct
func (p Parser) Act(instancePtr any) map[string][]ValidatorIssue {
	val := reflect.ValueOf(instancePtr).Elem()
	if val.Kind() != reflect.Struct {
		panic("value must be a struct")
	}

	t := val.Type()
	actors, exist := p.actors[t]
	if !exist {
		p.ParseActors(val.Interface())
		actors = p.actors[t]
	}

	issues := map[string][]ValidatorIssue{}
	for i, actor := range actors {
		field := val.Field(i)
		result := actor.Call([]reflect.Value{reflect.ValueOf(field)})
		fieldName := t.Field(i).Name
		issues[fieldName] = append(issues[fieldName], result[0].Interface().([]ValidatorIssue)...)
	}

	return issues
}
