package errors

import (
	"encoding/json"
	"fmt"
	"regexp"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
)

func TestErrors(t *testing.T) {

	t.Run("toErr", func(t *testing.T) {

		cases := []struct {
			desc     string
			reason   any
			expected []error
		}{
			{desc: "single_error", reason: fmt.Errorf("single error"), expected: []error{fmt.Errorf("single error")}},
			{desc: "many_errors", reason: []error{fmt.Errorf("err1"), fmt.Errorf("err2")}, expected: []error{fmt.Errorf("err1"), fmt.Errorf("err2")}},
			{desc: "many_parsed_errors", reason: []*Error{{reasons: []error{fmt.Errorf("err1")}}, {reasons: []error{fmt.Errorf("err1")}}}, expected: []error{&Error{reasons: []error{fmt.Errorf("err1")}}, &Error{reasons: []error{fmt.Errorf("err1")}}}},
			{
				desc: "any_slice",
				reason: []any{
					fmt.Errorf("err1"),
					"err2",
					&Error{reasons: []error{fmt.Errorf("err3")}},
					[]error{fmt.Errorf("err4"), fmt.Errorf("err5")},
				},
				expected: []error{
					fmt.Errorf("err1"),
					Reason{reason: "err2"},
					&Error{reasons: []error{fmt.Errorf("err3")}},
					fmt.Errorf("err4"),
					fmt.Errorf("err5"),
				},
			},
			{desc: "nil", reason: nil, expected: []error{nil}},
		}

		for _, c := range cases {
			c := c
			t.Run(c.desc, func(t *testing.T) {
				errs := toErr(c.reason)
				assert.Equal(t, c.expected, errs)
			})
		}
	})

	t.Run("New", func(t *testing.T) {
		err := New("reason")
		assert.Equal(t, []error{Reason{reason: "reason"}}, err.reasons)
		assert.Equal(t, 3, lo.CountBy(err.callers, func(c uintptr) bool {
			return c != 0
		}))
	})

	t.Run("NewSkip", func(t *testing.T) {
		err := NewSkip("reason", 3)
		assert.Equal(t, []error{Reason{reason: "reason"}}, err.reasons)
		assert.Equal(t, 2, lo.CountBy(err.callers, func(c uintptr) bool {
			return c != 0
		}))
	})

	t.Run("NewSkip_already_Error", func(t *testing.T) {
		err := New("reason")
		err2 := New(err)
		assert.Equal(t, err, err2)
	})

	t.Run("Accessors", func(t *testing.T) {
		err := New([]error{fmt.Errorf("a"), fmt.Errorf("b")})
		assert.Equal(t, 2, err.Len())
		assert.Equal(t, err.callers, err.Callers())
		assert.Equal(t, err.reasons, err.Unwrap())
	})

	t.Run("Error", func(t *testing.T) {
		cases := []struct {
			desc     string
			expected string
			reasons  []error
		}{
			{desc: "single", reasons: []error{fmt.Errorf("reason")}, expected: "reason"},
			{desc: "many", reasons: []error{fmt.Errorf("err1"), nil, fmt.Errorf("err2")}, expected: "err1\n<nil>\nerr2"},
			{desc: "empty_slice", reasons: []error{}, expected: "goyave.dev/goyave/util/errors.Error: the Error doesn't wrap any reason (empty reasons slice)"},
		}

		for _, c := range cases {
			c := c
			t.Run(c.desc, func(t *testing.T) {
				err := &Error{reasons: c.reasons}
				assert.Equal(t, c.expected, err.Error())
			})
		}
	})

	t.Run("String", func(t *testing.T) { // Note: this test is very sensitive to line changes (it checks line numbers in stacktraces). If you add lines before this, be sure to also update this test.
		emptySliceErr := New("")
		emptySliceErr.reasons = []error{}

		suberror := New("suberror")

		cases := []struct {
			expected *regexp.Regexp
			err      *Error
			desc     string
		}{
			{desc: "empty_slice", err: emptySliceErr, expected: regexp.MustCompile("^goyave.dev/goyave/util/errors.Error: the Error doesn't wrap any reason \\(empty reasons slice\\)\ngoyave\\.dev/goyave/v5/util/errors\\.TestErrors\\.func7\n\t(.*?)/goyave/util/errors/error_test\\.go:103\n")},
			{desc: "single", err: New("err1"), expected: regexp.MustCompile("^err1\ngoyave\\.dev/goyave/v5/util/errors\\.TestErrors\\.func7\n\t(.*?)/goyave/util/errors/error_test\\.go:114\n")},
			{
				desc:     "many",
				err:      New([]any{fmt.Errorf("err1"), "err2", nil, map[string]any{"key": "value"}, suberror}),
				expected: regexp.MustCompile("^err1\ngoyave\\.dev/goyave/v5/util/errors\\.TestErrors\\.func7\n\t(.*?)/goyave/util/errors/error_test\\.go:117\n([\\d\\S\\n\\t]*?)\n\nerr2\ngoyave\\.dev/goyave/v5/util/errors\\.TestErrors\\.func7\n\t(.*?)/goyave/util/errors/error_test\\.go:117\n([\\d\\S\\n\\t]*?)\n\n<nil>\ngoyave\\.dev/goyave/v5/util/errors\\.TestErrors\\.func7\n\t(.*?)/goyave/util/errors/error_test\\.go:117\n([\\d\\S\\n\\t]*?)\n\nmap\\[key:value\\]\ngoyave\\.dev/goyave/v5/util/errors\\.TestErrors\\.func7\n\t(.*?)/goyave/util/errors/error_test\\.go:117\n([\\d\\S\\n\\t]*?)\n\nsuberror\ngoyave\\.dev/goyave/v5/util/errors\\.TestErrors\\.func7\n\t(.*?)/goyave/util/errors/error_test\\.go:106\n([\\d\\S\\n\\t]*?)$"),
			},
			{desc: "single_already_error", err: New([]error{suberror}), expected: regexp.MustCompile("^suberror\ngoyave\\.dev/goyave/v5/util/errors\\.TestErrors\\.func7\n\t(.*?)/goyave/util/errors/error_test\\.go:106\n")},
		}

		for _, c := range cases {
			c := c
			t.Run(c.desc, func(t *testing.T) {
				assert.Regexp(t, c.expected, c.err.String())
			})
		}
	})

	t.Run("FileLine", func(t *testing.T) { // Note: this test is very sensitive to line changes. If you add lines before this, be sure to also update this test.

		cases := []struct {
			err      *Error
			expected *regexp.Regexp
			desc     string
		}{
			{desc: "OK", err: New(""), expected: regexp.MustCompile("/goyave/util/errors/error_test.go:138$")},
			{desc: "unknown", err: NewSkip("", 5), expected: regexp.MustCompile(`^\[unknown file line\]$`)}, // Skip more frames than necessary to have empty callers slice
		}

		for _, c := range cases {
			c := c
			t.Run(c.desc, func(t *testing.T) {
				assert.Regexp(t, c.expected, c.err.FileLine())
			})
		}
	})

	t.Run("JSON", func(t *testing.T) {
		emptySliceErr := New("")
		emptySliceErr.reasons = []error{}

		suberror := New("suberror")
		manySuberror := New([]error{fmt.Errorf("suberror1"), fmt.Errorf("suberror2")})

		cases := []struct {
			err         *Error
			desc        string
			expected    string
			expectedErr bool
		}{
			{desc: "empty_slice", err: emptySliceErr, expected: "\"goyave.dev/goyave/util/errors.Error: the Error doesn't wrap any reason (empty reasons slice)\""},
			{desc: "single", err: New(fmt.Errorf("error message")), expected: `"error message"`},
			{desc: "single_marshaler", err: New(map[string]any{"key": "value"}), expected: `{"key":"value"}`},
			{desc: "many", err: New([]any{nil, "ah", map[string]any{"key": "value"}, fmt.Errorf("error message"), suberror, manySuberror}), expected: `[null,"ah",{"key":"value"},"error message","suberror",["suberror1","suberror2"]]`},
			{desc: "marshal_unsupported_type", err: New(make(chan struct{})), expectedErr: true},
			{desc: "marshal_many_unsupported_type", err: New([]any{"a", make(chan struct{})}), expectedErr: true},
		}

		for _, c := range cases {
			c := c
			t.Run(c.desc, func(t *testing.T) {
				res, err := json.Marshal(c.err)
				if c.expectedErr {
					if !assert.Error(t, err) {
						return
					}
				} else if !assert.NoError(t, err) {
					return
				}
				assert.Equal(t, c.expected, string(res))
			})
		}
	})

	t.Run("Reason", func(t *testing.T) {
		reason := Reason{reason: map[string]any{"key": "value"}}
		assert.Equal(t, map[string]any{"key": "value"}, reason.Value())
		assert.Equal(t, "map[key:value]", reason.Error())

		res, err := reason.MarshalJSON()
		assert.NoError(t, err)
		assert.Equal(t, `{"key":"value"}`, string(res))

	})
}