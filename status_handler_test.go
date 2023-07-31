package goyave

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"goyave.dev/goyave/v5/config"
	"goyave.dev/goyave/v5/validation"
)

func prepareStatusHandlerTest() (*Request, *Response, *httptest.ResponseRecorder) {
	server, err := NewWithConfig(config.LoadDefault())
	if err != nil {
		panic(err)
	}
	httpReq := httptest.NewRequest(http.MethodGet, "/test", nil)
	req := NewRequest(httpReq)
	recorder := httptest.NewRecorder()
	resp := NewResponse(server, req, recorder)
	return req, resp, recorder
}

func TestPanicStatusHandler(t *testing.T) {
	t.Run("no_debug", func(t *testing.T) {
		req, resp, recorder := prepareStatusHandlerTest()
		resp.server.config.Set("app.debug", false)
		handler := &PanicStatusHandler{}
		handler.Init(resp.server)

		req.Extra[ExtraError] = fmt.Errorf("test error")
		handler.Handle(resp, req)
		res := recorder.Result()
		body, err := io.ReadAll(res.Body)
		assert.NoError(t, res.Body.Close())
		if !assert.NoError(t, err) {
			return
		}

		assert.Equal(t, "{\"error\":\"Internal Server Error\"}\n", string(body))
	})

	t.Run("debug", func(t *testing.T) {
		req, resp, recorder := prepareStatusHandlerTest()
		resp.server.config.Set("app.debug", true)
		logBuffer := &bytes.Buffer{}
		resp.server.ErrLogger = log.New(logBuffer, "", 0)
		handler := &PanicStatusHandler{}
		handler.Init(resp.server)

		// Don't wrap the error into a `*errors.Error` to check the handler
		// correctly calls `response.error()` and the error is converted to `*errors.Error`.
		req.Extra[ExtraError] = fmt.Errorf("test error")
		handler.Handle(resp, req)
		res := recorder.Result()
		body, err := io.ReadAll(res.Body)
		assert.NoError(t, res.Body.Close())
		if !assert.NoError(t, err) {
			return
		}

		assert.Equal(t, "{\"error\":\"test error\"}\n", string(body))

		// Error and stacktrace already printed by the recovery middleware or `response.Error`
		// (those are not executed in this test, thus leaving the log buffer empty)
		assert.Empty(t, logBuffer.String())
	})

	t.Run("nil_error", func(t *testing.T) {
		req, resp, recorder := prepareStatusHandlerTest()
		resp.server.config.Set("app.debug", true)
		logBuffer := &bytes.Buffer{}
		resp.server.ErrLogger = log.New(logBuffer, "", 0)
		handler := &PanicStatusHandler{}
		handler.Init(resp.server)

		handler.Handle(resp, req)
		res := recorder.Result()
		body, err := io.ReadAll(res.Body)
		assert.NoError(t, res.Body.Close())
		if !assert.NoError(t, err) {
			return
		}

		assert.Equal(t, "{\"error\":null}\n", string(body))

		// Error and stacktrace are not printed to console because recovery middleware
		// is not executed (no error raised, we just set the response status to 500 for example)
		assert.Empty(t, logBuffer.String())
	})
}

func TestErrorStatusHandler(t *testing.T) {
	req, resp, recorder := prepareStatusHandlerTest()
	handler := &ErrorStatusHandler{}
	handler.Init(resp.server)

	resp.Status(http.StatusNotFound)

	handler.Handle(resp, req)

	res := recorder.Result()
	body, err := io.ReadAll(res.Body)
	assert.NoError(t, res.Body.Close())
	if !assert.NoError(t, err) {
		return
	}

	assert.Equal(t, "{\"error\":\"Not Found\"}\n", string(body))
}

func TestValidationStatusHandler(t *testing.T) {
	req, resp, recorder := prepareStatusHandlerTest()
	handler := &ValidationStatusHandler{}
	handler.Init(resp.server)

	req.Extra[ExtraValidationError] = &validation.Errors{
		Errors: []string{"The body is required"},
		Fields: validation.FieldsErrors{
			"field": &validation.Errors{Errors: []string{"The field is required"}},
		},
	}
	req.Extra[ExtraQueryValidationError] = &validation.Errors{
		Fields: validation.FieldsErrors{
			"query": &validation.Errors{Errors: []string{"The query is required"}},
		},
	}

	handler.Handle(resp, req)

	res := recorder.Result()
	body, err := io.ReadAll(res.Body)
	assert.NoError(t, res.Body.Close())
	if !assert.NoError(t, err) {
		return
	}

	assert.Equal(t, "{\"error\":{\"body\":{\"fields\":{\"field\":{\"errors\":[\"The field is required\"]}},\"errors\":[\"The body is required\"]},\"query\":{\"fields\":{\"query\":{\"errors\":[\"The query is required\"]}}}}}\n", string(body))
}
