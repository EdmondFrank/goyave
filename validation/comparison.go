package validation

import (
	"fmt"

	"goyave.dev/goyave/v4/util/walk"
)

// ComparisonValidator validates the field under validation is greater than field identified
// by the given path. Mixed types are supported, meaning you can use this rule for the following (non-exhaustive) cases:
//   - Compare the length of two strings
//   - Compare the value of two numeric fields
//   - Compare a numeric field with the length of a string or a string length with a numeric field
//   - Compare a numeric field with the number of elements in an array
//   - Compare the number of keys in an object with a numeric field
//   - Compare a file (or multifile) size with a numeric field
type ComparisonValidator struct {
	BaseValidator
	Path *walk.Path
}

// Validate checks the field under validation satisfies this validator's criteria.
func (v *ComparisonValidator) validate(ctx *ContextV5, comparisonFunc func(size1, size2 float64) bool) bool {
	floatValue, isNumber := numberAsFloat64(ctx.Value)

	ok := true
	v.Path.Walk(ctx.Data, func(c *walk.Context) {
		if c.Path.Type == walk.PathTypeArray && c.Found == walk.ElementNotFound {
			return
		}

		if c.Found != walk.Found {
			ok = false
			c.Break()
			return
		}

		comparedFloatValue, isComparedNumber := numberAsFloat64(c.Value)
		if isNumber {
			if isComparedNumber {
				ok = floatValue > comparedFloatValue
			} else {
				ok = validateSizeV5(c.Value, func(size int) bool {
					return comparisonFunc(floatValue, float64(size))
				})
			}
		} else {
			if isComparedNumber {
				ok = validateSizeV5(ctx.Value, func(size int) bool {
					return comparisonFunc(float64(size), comparedFloatValue)
				})
			} else {
				ok = validateSizeV5(ctx.Value, func(size1 int) bool {
					return validateSizeV5(c.Value, func(size2 int) bool {
						return comparisonFunc(float64(size1), float64(size2))
					})
				})
			}
		}

		if !ok {
			c.Break()
		}
	})
	return ok
}

// IsTypeDependent returns true
func (v *ComparisonValidator) IsTypeDependent() bool { return true }

// MessagePlaceholders returns the ":other" placeholder.
func (v *ComparisonValidator) MessagePlaceholders(ctx *ContextV5) []string {
	return []string{
		":other", GetFieldName(v.Lang(), v.Path),
	}
}

//------------------------------

// GreaterThanValidator validates the field under validation is greater than the field identified
// by the given path. See `ComparisonValidator` for more details.
type GreaterThanValidator struct {
	ComparisonValidator
}

// Validate checks the field under validation satisfies this validator's criteria.
func (v *GreaterThanValidator) Validate(ctx *ContextV5) bool {
	return v.validate(ctx, func(size1, size2 float64) bool {
		return size1 > size2
	})
}

// Name returns the string name of the validator.
func (v *GreaterThanValidator) Name() string { return "greater_than" }

// GreaterThan validates the field under validation is greater than the field identified
// by the given path. Mixed types are supported, meaning you can use this rule for the following (non-exhaustive) cases:
//   - Compare the length of two strings
//   - Compare the value of two numeric fields
//   - Compare a numeric field with the length of a string or a string length with a numeric field
//   - Compare a numeric field with the number of elements in an array
//   - Compare the number of keys in an object with a numeric field
//   - Compare a file (or multifile) size with a numeric field
func GreaterThan(path string) *GreaterThanValidator {
	p, err := walk.Parse(path)
	if err != nil {
		panic(fmt.Errorf("validation.GreaterThan: path parse error: %w", err))
	}
	return &GreaterThanValidator{ComparisonValidator: ComparisonValidator{Path: p}}
}

//------------------------------

// GreaterThanEqualValidator validates the field under validation is greater than the field identified
// by the given path. See `ComparisonValidator` for more details.
type GreaterThanEqualValidator struct {
	ComparisonValidator
}

// Validate checks the field under validation satisfies this validator's criteria.
func (v *GreaterThanEqualValidator) Validate(ctx *ContextV5) bool {
	return v.validate(ctx, func(size1, size2 float64) bool {
		return size1 >= size2
	})
}

// Name returns the string name of the validator.
func (v *GreaterThanEqualValidator) Name() string { return "greater_than_equal" }

// GreaterThanEqual validates the field under validation is greater or equal to the field identified
// by the given path. Mixed types are supported, meaning you can use this rule for the following (non-exhaustive) cases:
//   - Compare the length of two strings
//   - Compare the value of two numeric fields
//   - Compare a numeric field with the length of a string or a string length with a numeric field
//   - Compare a numeric field with the number of elements in an array
//   - Compare the number of keys in an object with a numeric field
//   - Compare a file (or multifile) size with a numeric field
func GreaterThanEqual(path string) *GreaterThanEqualValidator {
	p, err := walk.Parse(path)
	if err != nil {
		panic(fmt.Errorf("validation.GreaterThanEqual: path parse error: %w", err))
	}
	return &GreaterThanEqualValidator{ComparisonValidator: ComparisonValidator{Path: p}}
}

//------------------------------

// LowerThanValidator validates the field under validation is lower than the field identified
// by the given path. See `ComparisonValidator` for more details.
type LowerThanValidator struct {
	ComparisonValidator
}

// Validate checks the field under validation satisfies this validator's criteria.
func (v *LowerThanValidator) Validate(ctx *ContextV5) bool {
	return v.validate(ctx, func(size1, size2 float64) bool {
		return size1 < size2
	})
}

// Name returns the string name of the validator.
func (v *LowerThanValidator) Name() string { return "lower_than" }

// LowerThan validates the field under validation is lower than the field identified
// by the given path. Mixed types are supported, meaning you can use this rule for the following (non-exhaustive) cases:
//   - Compare the length of two strings
//   - Compare the value of two numeric fields
//   - Compare a numeric field with the length of a string or a string length with a numeric field
//   - Compare a numeric field with the number of elements in an array
//   - Compare the number of keys in an object with a numeric field
//   - Compare a file (or multifile) size with a numeric field
func LowerThan(path string) *LowerThanValidator {
	p, err := walk.Parse(path)
	if err != nil {
		panic(fmt.Errorf("validation.LowerThan: path parse error: %w", err))
	}
	return &LowerThanValidator{ComparisonValidator: ComparisonValidator{Path: p}}
}

//------------------------------

// LowerThanEqualValidator validates the field under validation is lower or equal to the field identified
// by the given path. See `ComparisonValidator` for more details.
type LowerThanEqualValidator struct {
	ComparisonValidator
}

// Validate checks the field under validation satisfies this validator's criteria.
func (v *LowerThanEqualValidator) Validate(ctx *ContextV5) bool {
	return v.validate(ctx, func(size1, size2 float64) bool {
		return size1 <= size2
	})
}

// Name returns the string name of the validator.
func (v *LowerThanEqualValidator) Name() string { return "lower_than_equal" }

// LowerThanEqual validates the field under validation is lower or equal to the field identified
// by the given path. Mixed types are supported, meaning you can use this rule for the following (non-exhaustive) cases:
//   - Compare the length of two strings
//   - Compare the value of two numeric fields
//   - Compare a numeric field with the length of a string or a string length with a numeric field
//   - Compare a numeric field with the number of elements in an array
//   - Compare the number of keys in an object with a numeric field
//   - Compare a file (or multifile) size with a numeric field
func LowerThanEqual(path string) *LowerThanEqualValidator {
	p, err := walk.Parse(path)
	if err != nil {
		panic(fmt.Errorf("validation.LowerThanEqual: path parse error: %w", err))
	}
	return &LowerThanEqualValidator{ComparisonValidator: ComparisonValidator{Path: p}}
}