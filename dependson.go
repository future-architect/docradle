package docradle

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gookit/color"
	"golang.org/x/sync/errgroup"
)

// DependsOnCheckResult is a collection of dependency check
type DependsOnCheckResult struct {
	url      *url.URL
	headers  [][2]string
	timeout  time.Duration
	interval time.Duration
	duration time.Duration
	error    error
}

func (r DependsOnCheckResult) String() string {
	var builder strings.Builder
	builder.WriteString("  ")
	if r.Error() == nil {
		builder.WriteString("<bg=black;fg=green;op=reverse;>OK</> ")
	} else {
		builder.WriteString("<bg=black;fg=red;op=reverse;>NG</> ")
	}
	builder.WriteString("<blue>" + r.url.String() + "</> ")
	if r.Error() == nil {
		builder.WriteString("<gray>(wait " + r.duration.String() + ")</>")
	} else if errors.Is(r.error, context.DeadlineExceeded) {
		builder.WriteString("<red>Target service doesn't exist</>")
		builder.WriteString("<gray>(wait " + r.timeout.String() + ")</>")
	} else {
		builder.WriteString("<red>Error occured: " + r.error.Error() + "</>")
	}
	return builder.String()
}

func (r DependsOnCheckResult) Error() error {
	return r.error
}

// DumpAndSummaryDependsOnResult dumps depends-on check result
func DumpAndSummaryDependsOnResult(results []DependsOnCheckResult) LogOutputs {
	var outputs LogOutputs = make([]LogOutput, 0, len(results))
	for _, result := range results {
		outputs = append(outputs, LogOutput{
			Text:  result.String(),
			Error: result.Error() != nil,
		})
	}
	return outputs
}

func WaitForDependencies(ctx context.Context, dependsOns []DependsOn) (result []DependsOnCheckResult) {
	var eg *errgroup.Group
	eg, ctx = errgroup.WithContext(ctx)
	resultChan := make(chan DependsOnCheckResult)
	for _, dependsOn := range dependsOns {
		eg.Go(waitFor(ctx, dependsOn, resultChan))
	}
	result = make([]DependsOnCheckResult, len(dependsOns))
	for i := range dependsOns {
		result[i] = <-resultChan
	}
	return result
}

func waitFor(ctx context.Context, dependsOn DependsOn, resultChan chan<- DependsOnCheckResult) func() error {
	return func() error {
		u := dependsOn.URL
		var err error
		start := time.Now()
		switch u.Scheme {
		case "file":
			err = waitForFile(ctx, dependsOn)
		case "tcp", "tcp4", "tcp6", "unix":
			err = waitForSocket(ctx, dependsOn)
		case "http", "https":
			err = waitForHTTP(ctx, dependsOn)
		default:
			err = fmt.Errorf("invalid host protocol provided: %s. supported protocols are: tcp, tcp4, tcp6 and http", u.Scheme)
		}
		resultChan <- DependsOnCheckResult{
			url:      dependsOn.URL,
			headers:  dependsOn.Headers,
			timeout:  dependsOn.Timeout,
			interval: dependsOn.Interval,
			duration: time.Now().Sub(start),
			error:    err,
		}
		return nil
	}
}

func waitForFile(ctx context.Context, dependsOn DependsOn) error {
	ticker := time.NewTicker(dependsOn.Interval)
	defer ticker.Stop()
	ctx, cancel := context.WithTimeout(ctx, dependsOn.Timeout)
	defer cancel()
	for {
		if _, err := os.Stat(dependsOn.URL.Path); err == nil {
			return nil
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("File check error %s: %w", dependsOn.URL.String(), err)
		}
		select {
		case <-ctx.Done():
			if ctx.Err() == context.DeadlineExceeded {
				return fmt.Errorf("timeout for checking file '%s': %w", dependsOn.URL.String(), ctx.Err())
			}
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func waitForHTTP(ctx context.Context, dependsOn DependsOn) error {
	ticker := time.NewTicker(dependsOn.Interval)
	defer ticker.Stop()
	ctx, cancel := context.WithTimeout(ctx, dependsOn.Timeout)
	defer cancel()
	for {
		req, _ := http.NewRequest("HEAD", dependsOn.URL.String(), nil)
		req = req.WithContext(ctx)
		for _, header := range dependsOn.Headers {
			req.Header.Add(header[0], header[1])
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			if ctx.Err() == context.DeadlineExceeded {
				return fmt.Errorf("timeout for waiting web service '%s': %w", dependsOn.URL.String(), ctx.Err())
			}
		} else if err == nil && resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return nil
		}
		select {
		case <-ctx.Done():
			if ctx.Err() == context.DeadlineExceeded {
				return fmt.Errorf("timeout for checking server, '%s': %w", dependsOn.URL.String(), ctx.Err())
			}
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func waitForSocket(ctx context.Context, dependsOn DependsOn) error {
	u := dependsOn.URL
	finish := make(chan struct{})
	ctx, cancel := context.WithTimeout(ctx, dependsOn.Timeout)
	defer cancel()
	go func() {
		ticker := time.NewTicker(dependsOn.Interval)
		var host string
		if u.Scheme == "unix" {
			host = u.Path
		} else {
			host = u.Host
		}
		defer ticker.Stop()
		for {
			conn, _ := net.DialTimeout(u.Scheme, host, dependsOn.Interval)
			if conn != nil {
				conn.Close()
				finish <- struct{}{}
			}
			select {
			case <-ticker.C:
			case <-ctx.Done():
				return
			}
		}
	}()
	select {
	case <-ctx.Done():
		color.Printf("<red>Connection timeout:</> <yellow>%s</>\n", dependsOn.URL.String())
		return fmt.Errorf("timeout to connect %s://%s\n", u.Scheme, u.Host)
	case <-finish:
		return nil
	}
}
