package ffvm

import (
	"math"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

var (
	// singleton instance usage of Parser struct
	parser *Parser
)

func InitParser() {
	if parser != nil {
		return
	}

	parser = &Parser{
		mappers:    map[string]func(args ...string) func(any) any{},
		validators: map[string]func(args ...string) func(any) ValidatorIssue{},
		actors:     map[reflect.Type][]func(self any) (any, []ValidatorIssue){},
	}

	createInitialMappers()
	createInitialValidators()
}

func createInitialMappers() {
	parser.SetMapperGenerator("upper", func(args ...string) func(any) any {
		argCountPanic(args, 0, 0, "upper", Mapper)
		return func(self any) any {
			v, ok := self.(string)
			if !ok {
				return self
			}

			return strings.ToUpper(v)
		}
	})

	parser.SetMapperGenerator("lower", func(args ...string) func(any) any {
		argCountPanic(args, 0, 0, "lower", Validator)
		return func(self any) any {
			v, ok := self.(string)
			if !ok {
				return self
			}

			return strings.ToLower(v)
		}
	})

	parser.SetMapperGenerator("len", func(args ...string) func(any) any {
		argCountPanic(args, 0, 0, "len", Mapper)
		return func(self any) any {
			l := getLen(self)
			if l < 0 {
				return self
			}

			return l
		}
	})
}

func createInitialValidators() {
	parser.SetValidatorGenerator("not_nil", func(args ...string) func(any) ValidatorIssue {
		argCountPanic(args, 0, 0, "not_nil", Validator)
		return func(self any) ValidatorIssue {
			if self == nil {
				return CreateDefaultIssue("expected not nil value")
			}

			return NoIssue
		}
	})

	parser.SetValidatorGenerator("nil", func(args ...string) func(any) ValidatorIssue {
		argCountPanic(args, 0, 0, "nil", Validator)
		return func(self any) ValidatorIssue {
			if self != nil {
				return CreateDefaultIssue("expected nil value but, ", self)
			}

			return NoIssue
		}
	})

	parser.SetValidatorGenerator("empty", func(args ...string) func(any) ValidatorIssue {
		argCountPanic(args, 0, 0, "empty", Validator)
		return func(self any) ValidatorIssue {
			if !isEmpty(self) {
				return CreateDefaultIssue("expected empty value but was ", self)
			}

			return NoIssue
		}
	})

	parser.SetValidatorGenerator("not_empty", func(args ...string) func(any) ValidatorIssue {
		argCountPanic(args, 0, 0, "empty", Validator)
		return func(self any) ValidatorIssue {
			if isEmpty(self) {
				return CreateDefaultIssue("expected not empty value")
			}

			return NoIssue
		}
	})

	// create an alias for not_empty validator
	parser.validators["required"] = parser.validators["not_empty"]

	parser.SetValidatorGenerator("upper", func(args ...string) func(any) ValidatorIssue {
		argCountPanic(args, 0, 0, "upper", Validator)
		return func(self any) ValidatorIssue {
			if v, ok := self.(string); ok && self != strings.ToUpper(v) {
				return CreateDefaultIssue("expected upper case")
			}

			return NoIssue
		}
	})

	parser.SetValidatorGenerator("lower", func(args ...string) func(any) ValidatorIssue {
		argCountPanic(args, 0, 0, "lower", Validator)
		return func(self any) ValidatorIssue {
			if v, ok := self.(string); ok && v != strings.ToLower(v) {
				return CreateDefaultIssue("expected lower case string, but found ", self)
			}

			return NoIssue
		}
	})

	parser.SetValidatorGenerator("max_len", func(args ...string) func(any) ValidatorIssue {
		argCountPanic(args, 1, 1, "max_len", Validator)
		maxLen, err := strconv.ParseInt(args[0], 10, 0)
		if err != nil {
			panic("expected argument to be a valid integer, err=" + err.Error())
		}

		return func(self any) ValidatorIssue {
			l := getLen(self)
			if l < 0 || l <= int(maxLen) {
				return NoIssue
			}

			return CreateDefaultIssue("max len exceeded, max_len=", args[0], " but it len=", l)
		}
	})

	parser.SetValidatorGenerator("min_len", func(args ...string) func(any) ValidatorIssue {
		argCountPanic(args, 1, 1, "min_len", Validator)
		minLen, err := strconv.ParseInt(args[0], 10, 0)
		if err != nil {
			panic("expected argument to be a valid integer, err=" + err.Error())
		}

		return func(self any) ValidatorIssue {
			l := getLen(self)
			if l < 0 || l >= int(minLen) {
				return NoIssue
			}

			return CreateDefaultIssue("max_len=", args[0], " but it len=", l)
		}
	})

	parser.SetValidatorGenerator("len", func(args ...string) func(any) ValidatorIssue {
		argCountPanic(args, 1, 1, "len", Validator)
		expectedLen, err := strconv.ParseInt(args[0], 10, 0)
		if err != nil {
			panic("expected argument to be a valid integer, err=" + err.Error())
		}

		return func(self any) ValidatorIssue {
			l := getLen(self)
			if l < 0 || l == int(expectedLen) {
				return NoIssue
			}

			return CreateDefaultIssue("expected len to be ", args[0], " but len=", l)
		}
	})

	parser.SetValidatorGenerator("max", func(args ...string) func(any) ValidatorIssue {
		argCountPanic(args, 1, 1, "max", Validator)
		max, err := strconv.ParseFloat(args[0], 64)
		if err != nil {
			panic("expected argument to be a valid float, err=" + err.Error())
		}

		return func(self any) ValidatorIssue {
			if v, ok := getAsFloat64(self); !ok || v <= max {
				return NoIssue
			}

			return CreateDefaultIssue("expected not to be more than ", max, " but it was ", self)
		}
	})

	parser.SetValidatorGenerator("min", func(args ...string) func(any) ValidatorIssue {
		argCountPanic(args, 1, 1, "min", Validator)
		min, err := strconv.ParseFloat(args[0], 64)
		if err != nil {
			panic("expected argument to be a valid float, err=" + err.Error())
		}

		return func(self any) ValidatorIssue {
			if v, ok := getAsFloat64(self); !ok || v >= min {
				return NoIssue
			}

			return CreateDefaultIssue("expected not to be less than ", min, " but it was ", self)
		}
	})

	parser.SetValidatorGenerator("regex", func(args ...string) func(any) ValidatorIssue {
		argCountPanic(args, 1, 1, "regex", Validator)
		regex, err := regexp.Compile(args[0])
		if err != nil {
			panic("expected argument to be a valid regex, err=" + err.Error())
		}

		return func(self any) ValidatorIssue {
			if v, ok := self.(string); !ok || regex.MatchString(v) {
				return NoIssue
			}

			return CreateDefaultIssue("expected match string with regex=", args[0])
		}
	})

	parser.SetValidatorGenerator("enum", func(args ...string) func(any) ValidatorIssue {
		argCountPanic(args, 1, math.MaxInt, "regex", Validator)
		return func(self any) ValidatorIssue {
			for _, vv := range args {
				if self == vv {
					return NoIssue
				}
			}

			return CreateDefaultIssue("expected valid enum, be: ", self, ", expected one of: ", args)
		}
	})
}

// GetParser returns the singleton Parser instance, it will initialize it once called first time
func GetParser() *Parser {
	// do once, creation of parser
	if parser == nil {
		InitParser()
	}

	return parser
}

// Validate call act of the parser, which will run all mappers then validators and return validation issues
func Validate(val any) []ValidatorIssue {
	_, issues := GetParser().act(val, "")
	return issues
}
