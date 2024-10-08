package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.infratographer.com/permissions-api/internal/testingx"
)

func TestErrorMiddleware(t *testing.T) {
	ctx := context.Background()

	e := echo.New()
	e.Debug = true

	e.Use(echoTestLogger(t, e))
	e.Use(errorMiddleware)

	e.GET("/test", func(c echo.Context) error {
		errType := c.QueryParam("error")

		select {
		case <-c.Request().Context().Done():
			err := c.Request().Context().Err()

			switch errType {
			case "":
			case "echo":
				return echo.NewHTTPError(http.StatusInternalServerError, "some message").WithInternal(err)
			}

			return err
		case <-time.After(time.Second):
		}

		switch errType {
		case "":
		case "echo":
			return echo.ErrTeapot
		case "other":
			return io.ErrUnexpectedEOF
		case "internalCancel":
			return echo.NewHTTPError(http.StatusInternalServerError, "service error").WithInternal(context.Canceled)
		}

		return nil
	})

	type testinput struct {
		path  string
		delay time.Duration
	}

	testCases := []testingx.TestCase[testinput, *httptest.ResponseRecorder]{
		{
			Name: "NotCanceled",
			Input: testinput{
				path: "/test",
			},
			CheckFn: func(_ context.Context, t *testing.T, res testingx.TestResult[*httptest.ResponseRecorder]) {
				require.NoError(t, res.Err)
				require.NotNil(t, res.Success)

				assert.Equal(t, http.StatusOK, res.Success.Code)
			},
		},
		{
			Name: "EchoError",
			Input: testinput{
				path: "/test?error=echo",
			},
			CheckFn: func(_ context.Context, t *testing.T, res testingx.TestResult[*httptest.ResponseRecorder]) {
				require.NoError(t, res.Err)
				require.NotNil(t, res.Success)

				assert.Equal(t, http.StatusTeapot, res.Success.Code)
			},
		},
		{
			Name: "OtherError",
			Input: testinput{
				path: "/test?error=other",
			},
			CheckFn: func(_ context.Context, t *testing.T, res testingx.TestResult[*httptest.ResponseRecorder]) {
				require.NoError(t, res.Err)
				require.NotNil(t, res.Success)

				assert.Equal(t, http.StatusInternalServerError, res.Success.Code)
			},
		},
		{
			Name: "Canceled",
			Input: testinput{
				path:  "/test",
				delay: time.Second / 2,
			},
			CheckFn: func(_ context.Context, t *testing.T, res testingx.TestResult[*httptest.ResponseRecorder]) {
				require.NoError(t, res.Err)
				require.NotNil(t, res.Success)

				assert.Equal(t, http.StatusUnprocessableEntity, res.Success.Code)
			},
		},
		{
			Name: "Canceled echo",
			Input: testinput{
				path:  "/test?error=echo",
				delay: time.Second / 2,
			},
			CheckFn: func(_ context.Context, t *testing.T, res testingx.TestResult[*httptest.ResponseRecorder]) {
				require.NoError(t, res.Err)
				require.NotNil(t, res.Success)

				require.Equal(t, http.StatusUnprocessableEntity, res.Success.Code)

				resp := map[string]string{}

				err := json.Unmarshal(res.Success.Body.Bytes(), &resp)
				require.NoError(t, err, "no error expected decoding response body")

				expect := map[string]string{
					"error":   "code=422, message=some message, internal=context canceled",
					"message": "some message",
				}

				assert.Equal(t, expect, resp, "unexpected response")
			},
		},
		{
			Name: "Canceled echo internal",
			Input: testinput{
				path: "/test?error=internalCancel",
			},
			CheckFn: func(_ context.Context, t *testing.T, res testingx.TestResult[*httptest.ResponseRecorder]) {
				require.NoError(t, res.Err)
				require.NotNil(t, res.Success)

				require.Equal(t, http.StatusInternalServerError, res.Success.Code)

				resp := map[string]string{}

				err := json.Unmarshal(res.Success.Body.Bytes(), &resp)
				require.NoError(t, err, "no error expected decoding response body")

				expect := map[string]string{
					"error":   "code=500, message=service error, internal=context canceled",
					"message": "service error",
				}

				assert.Equal(t, expect, resp, "unexpected response")
			},
		},
	}

	testFn := func(ctx context.Context, input testinput) testingx.TestResult[*httptest.ResponseRecorder] {
		result := testingx.TestResult[*httptest.ResponseRecorder]{}

		ctx, cancel := context.WithCancel(ctx)

		defer cancel()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, input.path, nil)
		if err != nil {
			result.Err = err

			return result
		}

		resp := httptest.NewRecorder()

		if input.delay != 0 {
			go func() {
				time.Sleep(input.delay)

				cancel()
			}()
		}

		e.ServeHTTP(resp, req)

		result.Success = resp

		return result
	}

	testingx.RunTests(ctx, t, testCases, testFn)
}
