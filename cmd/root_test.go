package cmd

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func executeCommand(root *cobra.Command, args ...string) (output string, err error) {
	// Store the original value of stdout and stderr
	oldStdout := os.Stdout
	oldStderr := os.Stderr

	// Set the new values for stdout and stderr
	rOut, wOut, _ := os.Pipe()
	os.Stdout = wOut

	rErr, wErr, _ := os.Pipe()
	os.Stderr = wErr

	var wg sync.WaitGroup
	// Expect two routines to wait until they finish
	wg.Add(2)

	var stdoutBuf, stderrBuf bytes.Buffer

	go func() {
		// Tell waitGroup that this goroutine has finished
		defer wg.Done()
		io.Copy(&stdoutBuf, rOut)
	}()

	go func() {
		// Tell waitGroup that this goroutine has finished
		defer wg.Done()
		// Write data from the error reader into the variable stderrBurf
		io.Copy(&stderrBuf, rErr)
	}()

	root.SetArgs(args)
	err = root.Execute()

	// Stops writing to stdout and stderr
	wOut.Close()
	wErr.Close()

	os.Stdout = oldStdout
	os.Stderr = oldStderr

	// Wait for the goroutines to finish copying all output data to stdout and stderr
	wg.Wait()

	return stdoutBuf.String() + stderrBuf.String(), err
}

func newMockHTTPClient(statusCode int, body string) *http.Client {
	return &http.Client{
		Transport: roundTripFunc(func(req *http.Request) *http.Response {
			return &http.Response{
				StatusCode: statusCode,
				Body:       io.NopCloser(bytes.NewBufferString(body)),
				Header:     make(http.Header),
			}
		}),
	}
}

func newMockStreamingHTTPClient(statusCode int, chunks []string) *http.Client {
	return &http.Client{
		Transport: roundTripFunc(func(req *http.Request) *http.Response {
			var buf bytes.Buffer
			for _, chunk := range chunks {
				buf.WriteString("data: " + chunk + "\n\n")
			}

			// Tell caller end of stream
			buf.WriteString("data: [DONE]\n\n")

			return &http.Response{
				StatusCode: statusCode,
				Body:       io.NopCloser(&buf),
				Header:     make(http.Header),
			}
		}),
	}
}

type roundTripFunc func(req *http.Request) *http.Response

// The HTTP client transport require a function that implements the RoundTrip interface. This is just a mock that
// returns the provided req data. More like an identity?
func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

func TestMain(m *testing.M) {
	originalZerologGlobalLevel := zerolog.GlobalLevel()
	zerolog.SetGlobalLevel(zerolog.Disabled)

	code := m.Run()

	// Disable all kind of logging
	zerolog.SetGlobalLevel(originalZerologGlobalLevel)

	os.Exit(code)
}

func TestRootCommand(t *testing.T) {
	viper.Reset()
	cobra.OnInitialize()

	mockResponse := `{
        "id": "chat-compeltion-test",
        "choices": [
            {
                "index": 0,
                "message": {
                    "role": "assistant",
                    "content": "Because we are hardcore typing machines"
                }
            }
        ]
    }`

	originalHttpClient := httpClient
	defer func() {
		httpClient = originalHttpClient
	}()

	httpClient = newMockHTTPClient(http.StatusOK, mockResponse)

	viper.Set("api_key", "super_secret_key")

	t.Run("basic (streaming) prompt", func(t *testing.T) {
		streamingChunks := []string{
			`{"id": "chat-stream-test", "choices": [{"index": 0, "delta": {"content": "This"}}]}`,
			`{"id": "chat-stream-test", "choices": [{"index": 0, "delta": {"content": " is"}}]}`,
			`{"id": "chat-stream-test", "choices": [{"index": 0, "delta": {"content": " a"}}]}`,
			`{"id": "chat-stream-test", "choices": [{"index": 0, "delta": {"content": " streamed"}}]}`,
			`{"id": "chat-stream-test", "choices": [{"index": 0, "delta": {"content": " response."}}]}`,
			`{"id": "chat-stream-test", "choices": [{"index": 0, "delta": {}, "finish_reason": "stop"}]}`,
		}

		httpClient = newMockStreamingHTTPClient(http.StatusOK, streamingChunks)

		output, err := executeCommand(rootCmd, "--stream-mode", "Tell me a story.")
		if err != nil {
			t.Log("Error in executeCommand:", err)
			t.Fatalf("streaming command failed: %v", err)
		}

		expected := "This is a streamed response."
		if !strings.Contains(output, expected) {
			t.Log("Output:", output)
			t.Errorf("expected output to contain %q, but got %q", expected, output)
		}
	})

	t.Run("empty prompt", func(t *testing.T) {
		httpClient = newMockHTTPClient(http.StatusOK, mockResponse)

		_, err := executeCommand(rootCmd)

		if err == nil {
			t.Fatalf("Expected an error but got nil")
		}
	})

	t.Run("blocking prompt", func(t *testing.T) {
		httpClient = newMockHTTPClient(http.StatusOK, mockResponse)

		output, err := executeCommand(rootCmd, "--stream-mode=false", "Why do we use Golang instead of Rust?")
		if err != nil {
			t.Fatalf("root command failed: %v", err)
		}

		expected := "Because we are hardcore typing machines"
		if !strings.Contains(output, expected) {
			t.Errorf("expected output to contain %q, but got %q", expected, output)
		}
	})
}

func TestModelsCommand(t *testing.T) {
	viper.Reset()
	cobra.OnInitialize(initConfig)

	rootCmd.AddCommand(ModelsCmd)

	t.Run("list models", func(test *testing.T) {
		output, err := executeCommand(rootCmd, "models")
		if err != nil {
			test.Fatalf("models command failed: %v", err)
		}

		expected := "Available models:\ngpt-3.5-turbo\ngpt-4"
		if !strings.Contains(output, expected) {
			test.Errorf("expected output to contain %q, but got %q", expected, output)
		}
	})
}
