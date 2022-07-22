package validation

import (
	"reflect"
	"time"

	"gorm.io/gorm"
	"goyave.dev/goyave/v4/config"
	"goyave.dev/goyave/v4/lang"
	"goyave.dev/goyave/v4/util/maputil"
	"goyave.dev/goyave/v4/util/walk"
)

type Options struct {
	Data      any
	Rules     RulerV5
	IsJSON    bool
	Languages *lang.Languages
	DB        *gorm.DB
	Config    *config.Config
	Lang      string
	Extra     map[string]any
}

type ContextV5 struct {
	Options *Options
	Data    any
	Extra   map[string]any
	Value   any
	Parent  any
	Field   *FieldV5
	Rule    Validator
	Now     time.Time

	// The name of the field under validation
	Name string

	errors []error
}

// DB get the database instance given through the validation Options.
// Panics if there is none.
func (c *ContextV5) DB() *gorm.DB {
	if c.Options.DB == nil {
		panic("DB is not set in validation options")
	}
	return c.Options.DB
}

// Config get the configuration given through the validation Options.
// Panics if there is none.
func (c *ContextV5) Config() *config.Config {
	if c.Options.Config == nil {
		panic("Config is not set in validation options")
	}
	return c.Options.Config
}

// AddError adds an error to the validation context. This is NOT supposed
// to be used when the field under validation doesn't match the rule, but rather
// when there has been an operation error (such as a database error).
func (c *ContextV5) AddError(err error) {
	c.errors = append(c.errors, err)
}

type validator struct {
	validationErrors *ErrorsV5
	options          *Options
	now              time.Time
	errors           []error
}

func ValidateV5(options *Options) (*ErrorsV5, []error) {
	validator := &validator{
		options:          options,
		now:              time.Now(),
		errors:           []error{},
		validationErrors: &ErrorsV5{},
	}

	rules := options.Rules.AsRules()
	for _, fieldName := range rules.sortedKeys {
		field := rules.Fields[fieldName]
		if fieldName == CurrentElement {
			// Validate the root element
			// TODO there may be a cleaner way to do this?
			fakeParent := map[string]any{CurrentElement: options.Data}
			validator.validateField(fieldName, field, fakeParent, nil)
			options.Data = fakeParent[CurrentElement]
		} else {
			validator.validateField(fieldName, field, options.Data, nil)
		}
	}
	// TODO PostValidationHooksV5
	// for _, hook := range rules.PostValidationHooks {
	// 	errors = hook(data, errors, now)
	// }

	if len(validator.errors) != 0 {
		return nil, validator.errors
	}
	if len(validator.validationErrors.Errors) != 0 || len(validator.validationErrors.Elements) != 0 || len(validator.validationErrors.Fields) != 0 {
		return validator.validationErrors, nil
	}
	return nil, nil
}

func (v *validator) validateField(fieldName string, field *FieldV5, walkData any, parentPath *walk.Path) {
	field.Path.Walk(walkData, func(c walk.Context) {
		parentObject, parentIsObject := c.Parent.(map[string]any)
		if parentIsObject && !field.IsNullable() && c.Value == nil {
			delete(parentObject, c.Name)
		}

		if shouldConvertSingleValueArray(fieldName, v.options.IsJSON) && c.Found == walk.Found {
			c.Value = v.convertSingleValueArray(field, c.Value, parentObject) // Convert single value arrays in url-encoded requests
			parentObject[c.Name] = c.Value
		}

		if v.isAbsent(field, c, v.options.Data) {
			return
		}

		if field.Elements != nil {
			// This is an array, recursively validate it so it can be converted to correct type
			if _, ok := c.Value.([]any); !ok {
				if newValue, ok := makeGenericSlice(c.Value); ok {
					replaceValue(c.Value, c)
					c.Value = newValue
				}
			}

			path := c.Path
			if parentPath != nil {
				clone := parentPath.Clone()
				tail := clone.Tail()
				tail.Type = walk.PathTypeArray
				tail.Index = &c.Index
				tail.Next = path.Next
				path = clone
			}
			v.validateField(fieldName+"[]", field.Elements, c.Value, path)
		}

		value := c.Value
		for _, rule := range field.Rules {
			if _, ok := rule.(*Nullable); ok {
				if value == nil {
					break
				}
				continue
			}

			ctx := &ContextV5{
				Options: v.options,
				Data:    v.options.Data,
				Extra:   maputil.Clone(v.options.Extra),
				Value:   value,
				Parent:  c.Parent,
				Field:   field,
				Rule:    rule,
				Now:     v.now,
				Name:    c.Name,
			}
			if !rule.Validate(ctx) {
				path := field.getErrorPath(parentPath, c)
				// message := processPlaceholders(fieldName, v.getMessage(field, rule, reflect.ValueOf(value)), language, ctx)
				// TODO placeholderV5
				message := v.getMessage(field, rule, reflect.ValueOf(value))
				if fieldName == CurrentElement {
					v.validationErrors.Add(path, message)
				} else {
					v.validationErrors.Add(&walk.Path{Type: walk.PathTypeObject, Next: path}, message)
				}
				continue
			}

			value = ctx.Value
		}
		// Value may be modified (converting rule), replace it in the parent element
		replaceValue(value, c)
	})
}

func (v *validator) convertSingleValueArray(field *FieldV5, value any, data map[string]any) any {
	rv := reflect.ValueOf(value)
	kind := rv.Kind().String()
	if field.IsArray() && kind != "slice" {
		rt := reflect.TypeOf(value)
		slice := reflect.MakeSlice(reflect.SliceOf(rt), 0, 1)
		slice = reflect.Append(slice, rv)
		return slice.Interface()
	}
	return value
}

func (v *validator) isAbsent(field *FieldV5, c walk.Context, data any) bool {
	if c.Found == walk.ParentNotFound {
		return true
	}
	// requiredCtx := &ContextV5{
	// 	Options: v.options,
	// 	Data:   data,
	// 	Value:  c.Value,
	// 	Parent: c.Parent,
	// 	Field:  field,
	// 	Rule:   &Rule{Name: "required"},
	// 	Name:   c.Name,
	// }
	// return !field.IsRequired() && !validateRequired(requiredCtx)
	// TODO use validateRequiredV5
	return !field.IsRequired() && !(!field.IsNullable() && c.Value == nil)
}

func (v *validator) getMessage(field *FieldV5, rule Validator, value reflect.Value) string {
	langEntry := "validation.rules." + rule.Name()
	if rule.IsTypeDependent() {
		expectedType := v.findTypeRule(field.Rules)
		if expectedType == "unsupported" {
			langEntry += "." + getFieldType(value)
		} else {
			if expectedType == "integer" {
				expectedType = "numeric"
			}
			langEntry += "." + expectedType
		}
	}

	lastParent := field.Path.LastParent()
	if lastParent != nil && lastParent.Type == walk.PathTypeArray {
		langEntry += ".array"
	}

	return v.options.Languages.Get(v.options.Lang, langEntry)
}

// findTypeRule find the expected type of a field for a given array dimension.
func (v *validator) findTypeRule(rules []Validator) string {
	for _, rule := range rules {
		if rule.IsType() {
			return rule.Name()
		}
	}
	return "unsupported"
}