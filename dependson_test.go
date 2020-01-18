package docradle

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
)

func mustUrlParse(t *testing.T, s string) *url.URL {
	t.Helper()
	u, err := url.Parse(s)
	if err != nil {
		panic(err)
	}
	return u
}

func TestWaitForDependencies_UnknownSchema(t *testing.T) {
	results := WaitForDependencies(context.Background(), []DependsOn{
		{
			URL:      mustUrlParse(t, "unknown://"),
			Timeout:  time.Second,
			Interval: time.Second,
		},
	})
	assert.Len(t, results, 1)
	assert.Error(t, results[0].Error())
}

func toUrl(t *testing.T, scheme, filePath string) string {
	t.Helper()
	path := filepath.ToSlash(filePath)
	if filepath.IsAbs(filePath) && !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	u := &url.URL{
		Scheme: scheme,
		Path:   path,
	}
	return u.String()
}

func TestWaitForDependencies_File(t *testing.T) {
	testcases := []struct {
		name string
		ok   bool
	}{
		{
			name: "ok",
			ok:   true,
		},
		{
			name: "ng",
			ok:   false,
		},
	}
	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			dirPath := filepath.Join(os.TempDir(), "cradle"+xid.New().String())
			defer os.RemoveAll(dirPath)

			sampleFile := filepath.Join(dirPath, "test.txt")
			t.Log("filepath: ", sampleFile)

			if tt.ok {
				go func() {
					time.Sleep(time.Microsecond * 10)
					os.MkdirAll(dirPath, 0755)
					t.Log(ioutil.WriteFile(sampleFile, nil, 0644))
				}()
			}

			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			results := WaitForDependencies(ctx, []DependsOn{
				{
					URL:      mustUrlParse(t, toUrl(t, "file", sampleFile)),
					Timeout:  time.Millisecond * 15,
					Interval: time.Millisecond * 5,
				},
			})
			assert.Len(t, results, 1)
			if tt.ok {
				assert.NoError(t, results[0].Error())
			} else {
				assert.Error(t, results[0].Error())
			}
		})
	}
}

func TestWaitForDependencies_TCP(t *testing.T) {
	testcases := []struct {
		name    string
		network string
		port    string
		host    string
		ok      bool
	}{
		{
			name:    "tcp ok",
			network: "tcp",
			port:    ":19876",
			host:    "127.0.0.1",
			ok:      true,
		},
		{
			name:    "tcp ng",
			network: "tcp",
			port:    ":19877",
			host:    "127.0.0.1",
			ok:      false,
		},
		{
			name:    "tcp4 ok",
			network: "tcp4",
			port:    ":19878",
			host:    "127.0.0.1",
			ok:      true,
		},
		{
			name:    "tcp4 ng",
			network: "tcp4",
			port:    ":19879",
			host:    "127.0.0.1",
			ok:      false,
		},
		{
			name:    "tcp6 ok",
			network: "tcp6",
			port:    ":19880",
			host:    "[::1]",
			ok:      true,
		},
		{
			name:    "tcp6 ng",
			network: "tcp6",
			port:    ":19881",
			host:    "[::1]",
			ok:      false,
		},
	}

	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			var listener net.Listener
			defer func() {
				if listener != nil {
					listener.Close()
				}
			}()
			if tt.ok {
				go func() {
					t.Helper()
					var err error
					listener, err = net.Listen(tt.network, tt.port)
					assert.NoError(t, err)
				}()
			}

			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			results := WaitForDependencies(ctx, []DependsOn{
				{
					URL:      mustUrlParse(t, fmt.Sprintf("%s://%s%s", tt.network, tt.host, tt.port)),
					Timeout:  time.Millisecond * 15,
					Interval: time.Millisecond * 5,
				},
			})
			assert.Len(t, results, 1)
			if tt.ok {
				assert.NoError(t, results[0].Error())
			} else {
				assert.Error(t, results[0].Error())
			}
		})
	}
}

func TestWaitForDependencies_UnixDomainSocket(t *testing.T) {
	testcases := []struct {
		name string
		ok   bool
	}{
		{
			name: "ok",
			ok:   true,
		},
		{
			name: "ng",
			ok:   false,
		},
	}
	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			var listener net.Listener
			defer func() {
				if listener != nil {
					listener.Close()
				}
			}()
			dirPath := filepath.Join(os.TempDir(), "cradle"+xid.New().String())
			defer os.RemoveAll(dirPath)

			sampleSocket := filepath.Join(dirPath, "socket")
			t.Log("filepath: ", sampleSocket)

			if tt.ok {
				go func() {
					time.Sleep(time.Microsecond * 10)
					os.MkdirAll(dirPath, 0755)
					var err error
					listener, err = net.Listen("unix", sampleSocket)
					assert.NoError(t, err)
				}()
			}

			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			results := WaitForDependencies(ctx, []DependsOn{
				{
					URL:      mustUrlParse(t, toUrl(t, "unix", sampleSocket)),
					Timeout:  time.Millisecond * 15,
					Interval: time.Millisecond * 5,
				},
			})
			assert.Len(t, results, 1)
			if tt.ok {
				assert.NoError(t, results[0].Error())
			} else {
				assert.Error(t, results[0].Error())
			}
		})
	}
}

func TestWaitForDependencies_HTTP(t *testing.T) {
	testcases := []struct {
		name              string
		url               string
		port              string
		headers           [][2]string
		headerShouldMatch []string
		ok                bool
	}{
		{
			name: "http ok",
			url:  "http://localhost:19883/health",
			port: ":19883",
			ok:   true,
		},
		{
			name: "http wrong port",
			url:  "http://localhost:19884/notfound",
			port: ":29884",
			ok:   false,
		},
		{
			name: "http not found",
			url:  "http://localhost:19885/notfound",
			port: ":19885",
			ok:   false,
		},
		{
			name:              "http header match",
			url:               "http://localhost:19886/health",
			port:              ":19886",
			headers:           [][2]string{{"Authorization", "Bearer abcdefghi"}},
			headerShouldMatch: []string{"Authorization", "Bearer abcdefghi"},
			ok:                true,
		},
		{
			name:              "http header not match",
			url:               "http://localhost:19888/health",
			port:              ":19888",
			headers:           [][2]string{},
			headerShouldMatch: []string{"Authorization", "Bearer abcdefghi"},
			ok:                false,
		},
	}

	for _, tt := range testcases {
		t.Run(tt.name, func(t *testing.T) {
			var server *http.Server
			defer func() {
				server.Close()
			}()
			go func() {
				t.Helper()
				time.Sleep(10 * time.Millisecond)
				port := tt.port
				server = &http.Server{
					Addr: port,
					Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						if r.URL.Path != "/health" {
							w.WriteHeader(http.StatusNotFound)
							return
						}
						if len(tt.headerShouldMatch) > 0 {
							if r.Header.Get(tt.headerShouldMatch[0]) != tt.headerShouldMatch[1] {
								w.WriteHeader(http.StatusForbidden)
								return
							}
						}
						w.WriteHeader(http.StatusOK)
					}),
				}
				server.ListenAndServe()
			}()

			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			results := WaitForDependencies(ctx, []DependsOn{
				{
					URL:      mustUrlParse(t, tt.url),
					Headers:  tt.headers,
					Timeout:  time.Millisecond * 30,
					Interval: time.Millisecond * 5,
				},
			})
			assert.Len(t, results, 1)
			if tt.ok {
				assert.NoError(t, results[0].Error())
			} else {
				assert.Error(t, results[0].Error())
			}
		})
	}
}
