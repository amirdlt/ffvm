package ffvm

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

func newParser() Parser {
	p := Parser{
		mappers:    map[string]func(in ...reflect.Value) reflect.Value{},
		validators: map[string]func(val any, args ...string) ValidatorIssue{},
		actors:     map[reflect.Type][]reflect.Value{},
	}

	p.setMapperFunc("upper", func(str string) string {
		return strings.ToUpper(str)
	})

	p.setMapperFunc("lower", func(str string) string {
		return strings.ToLower(str)
	})

	p.setMapperFunc("len", func(v any) int { return reflect.ValueOf(v).Len() })

	p.setValidatorFunc("not_nil", func(v any) ValidatorIssue {
		if v == nil {
			return ValidatorIssue{
				Issue: "expected not nil value",
			}
		}

		return ValidatorIssue{}
	})

	p.setValidatorFunc("nil", func(v any) ValidatorIssue {
		if v != nil {
			return ValidatorIssue{
				Issue: "expected nil value but, " + fmt.Sprint(v),
			}
		}

		return ValidatorIssue{}
	})

	p.setValidatorFunc("empty", func(v any) ValidatorIssue {
		if !reflect.DeepEqual(reflect.ValueOf(v), reflect.Zero(reflect.TypeOf(v))) {
			return ValidatorIssue{
				Issue: "expected empty value but, " + fmt.Sprint(v),
			}
		}

		return ValidatorIssue{}
	})

	p.setValidatorFunc("not_empty", func(v any) ValidatorIssue {
		if reflect.DeepEqual(reflect.ValueOf(v), reflect.Zero(reflect.TypeOf(v))) {
			return ValidatorIssue{
				Issue: "expected not empty value",
			}
		}

		return ValidatorIssue{}
	})

	// create an alias for not_empty validator
	p.validators["required"] = p.validators["not_empty"]

	p.setValidatorFunc("upper", func(v string) ValidatorIssue {
		if v != strings.ToUpper(v) {
			return ValidatorIssue{
				Issue: "expected upper case",
			}
		}

		return ValidatorIssue{}
	})

	p.setValidatorFunc("lower", func(v string) ValidatorIssue {
		if v != strings.ToLower(v) {
			return ValidatorIssue{
				Issue: "expected lower case",
			}
		}

		return ValidatorIssue{}
	})

	p.setValidatorFunc("max_len", func(v any, maxLen string) ValidatorIssue {
		_maxLen, err := strconv.ParseInt(maxLen, 10, 0)
		if err != nil {
			panic(err)
		}

		_len := reflect.ValueOf(v).Len()
		if _len > int(_maxLen) {
			return ValidatorIssue{
				Issue: "max len exceeded, expected max len " + maxLen + " but is " + fmt.Sprint(_len),
			}
		}

		return ValidatorIssue{}
	})

	p.setValidatorFunc("min_len", func(v any, minLen string) ValidatorIssue {
		_minLen, err := strconv.ParseInt(minLen, 10, 0)
		if err != nil {
			panic(err)
		}

		_len := reflect.ValueOf(v).Len()
		if _len < int(_minLen) {
			return ValidatorIssue{
				Issue: "less than min len, expected min len " + minLen + " but is " + fmt.Sprint(_len),
			}
		}

		return ValidatorIssue{}
	})

	p.setValidatorFunc("len", func(v any, expectedLen string) ValidatorIssue {
		_expectedLen, err := strconv.ParseInt(expectedLen, 10, 0)
		if err != nil {
			panic(err)
		}

		_len := reflect.ValueOf(v).Len()
		if _len != int(_expectedLen) {
			return ValidatorIssue{
				Issue: "expected len to be " + expectedLen + " but is " + fmt.Sprint(_len),
			}
		}

		return ValidatorIssue{}
	})

	p.setValidatorFunc("max", func(v any, max string) ValidatorIssue {
		_max, err := strconv.ParseFloat(max, 64)
		if err != nil {
			panic(err)
		}

		if val, err := strconv.ParseFloat(fmt.Sprint(v), 64); err != nil {
			panic(err)
		} else if val > _max {
			return ValidatorIssue{
				Issue: "expected not to be more than " + max + " but be " + fmt.Sprint(val),
			}
		}

		return ValidatorIssue{}
	})

	p.setValidatorFunc("min", func(v any, min string) ValidatorIssue {
		_min, err := strconv.ParseFloat(min, 64)
		if err != nil {
			panic(err)
		}

		if val, err := strconv.ParseFloat(fmt.Sprint(v), 64); err != nil {
			panic(err)
		} else if val < _min {
			return ValidatorIssue{
				Issue: "expected not to be less than " + min + " but be " + fmt.Sprint(val),
			}
		}

		return ValidatorIssue{}
	})

	p.setValidatorFunc("regex", func(v string, pattern string) ValidatorIssue {
		if matched, err := regexp.MatchString(pattern, v); err != nil {
			panic(err)
		} else if !matched {
			return ValidatorIssue{
				Issue: "expected match string with " + pattern,
			}
		}

		return ValidatorIssue{}
	})

	p.setValidatorFunc("enum", func(v any, values ...any) {
		for _, vv := range values {
			if reflect.DeepEqual(v, vv) {
				return ValidatorIssue{}
			}
		}

		return ValidatorIssue{
			Issue: "expected valid enum, be: " + fmt.Sprint(v) + ", expected one of: " + fmt.Sprint(values)
		}
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
	// an arbitrary count of arguments, these args all
	// are just strings that must be parsed in the function itself
	// return a single ValidatorIssue struct
	// if no error has been detected an empty ValidatorIssue should be returned
	validators map[string]func(val any, args ...string) ValidatorIssue

	// each type has a list of actors
	// in the same order of reflect.Value.Field
	// each actor is a function that accept a value
	// then return the new value and the validation result
	actors map[reflect.Type][]reflect.Value
}

func (p Parser) setMapperFunc(funcName string, mapper any) {
	val := reflect.ValueOf(mapper)
	p.mappers[funcName] = func(in ...reflect.Value) reflect.Value {
		return val.Call(in)[0]
	}
}

func (p Parser) setValidatorFunc(funcName string, validator any) {
	val := reflect.ValueOf(validator)
	p.validators[funcName] = func(v any, args ...string) ValidatorIssue {
		_args := make([]reflect.Value, len(args))
		for i, arg := range args {
			_args[i] = reflect.ValueOf(arg)
		}

		return val.Call(append([]reflect.Value{reflect.ValueOf(v)}, _args...))[0].Interface().(ValidatorIssue)
	}
}

// parseActors parse all exported actors and save them in a map
func (p Parser) parseActors(instance any) {
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
			if !strings.Contains(tag, ",") {
				tag = "," + tag
			}

			tokens := strings.Split(tag, ",")
			mappers, validators := strings.Split(tokens[0], ";"), strings.Split(tokens[1], ";")
			actors = append(actors, reflect.ValueOf(func(val reflect.Value) (issues []ValidatorIssue) {
				for _, mapper := range mappers {
					if mapper == "" {
						continue
					}

					val.Set(p.mappers[mapper](val))
				}

				for _, validator := range validators {
					if validator == "" {
						continue
					}

					validator, args := tokenizeFunction(validator)
					issue := p.validators[validator](val.Interface(), args...)
					if issue.Issue == "" {
						continue
					} else if issue.Level == "" {
						issue.Level = "UNSET"
					}

					issues = append(issues, issue)
				}

				return issues
			}))
		} else {
			actors = append(actors, reflect.ValueOf(func(val reflect.Value) (issues []ValidatorIssue) {
				return []ValidatorIssue{}
			}))
		}
	}

	p.actors[t] = actors
}

// act firstly do the manipulation, then validation
// instancePtr is a pointer to an instance of the struct
func (p Parser) act(instancePtr any) map[string][]ValidatorIssue {
	val := reflect.ValueOf(instancePtr).Elem()
	if val.Kind() != reflect.Struct {
		panic("value must be a struct")
	}

	t := val.Type()
	actors, exist := p.actors[t]
	if !exist {
		p.parseActors(val.Interface())
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

func tokenizeFunction(f string) (string, []string) {
	index := strings.Index(f, "=")
	if index <= 0 {
		return f, nil
	}

	return f[:index], strings.Split(f[index+1:], "&")
}

var parser *Parser

func Validate(val any) map[string][]ValidatorIssue {
	if parser == nil {
		p := newParser()
		parser = &p
	}

	return parser.act(val)
}
