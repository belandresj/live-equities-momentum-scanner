package reference

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type referenceProvider string

const (
	universeProvider referenceProvider = "universe"
	priorProvider    referenceProvider = "prior_close"
)

func TestBothResolversBoundedAcquisitionDiagnostics(t *testing.T) {
	providers := []referenceProvider{universeProvider, priorProvider}
	for _, provider := range providers {
		t.Run(string(provider), func(t *testing.T) {
			for _, status := range []int{http.StatusRequestTimeout, http.StatusTooManyRequests, http.StatusServiceUnavailable} {
				t.Run(http.StatusText(status)+" retries", func(t *testing.T) {
					var attempts atomic.Int64
					server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
						if attempts.Add(1) == 1 {
							http.Error(writer, "bounded synthetic failure", status)
							return
						}
						_, _ = io.WriteString(writer, validProviderBody(provider))
					}))
					defer server.Close()
					diagnostics, err := resolveProvider(t, provider, context.Background(), server, nil)
					if err != nil || diagnostics.RequestCount != 1 || diagnostics.AttemptCount != 2 || diagnostics.TerminalReason != TerminalReasonNone {
						t.Fatalf("diagnostics = %+v, err = %v", diagnostics, err)
					}
					assertProviderPageCount(t, provider, diagnostics, 1)
				})
			}

			t.Run("response read failure retries", func(t *testing.T) {
				var attempts atomic.Int64
				server := groupedServer(validProviderBody(provider))
				defer server.Close()
				diagnostics, err := resolveProvider(t, provider, context.Background(), server, func(universe *Resolver, prior *PriorCloseResolver) {
					client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
						if attempts.Add(1) == 1 {
							return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(failingReader{})}, nil
						}
						return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(validProviderBody(provider)))}, nil
					})}
					universe.HTTPClient = client
					prior.HTTPClient = client
				})
				if err != nil || diagnostics.AttemptCount != 2 {
					t.Fatalf("diagnostics = %+v, err = %v", diagnostics, err)
				}
			})

			t.Run("permanent HTTP status is not retried", func(t *testing.T) {
				var attempts atomic.Int64
				server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
					attempts.Add(1)
					http.Error(writer, "permanent", http.StatusBadRequest)
				}))
				defer server.Close()
				diagnostics, err := resolveProvider(t, provider, context.Background(), server, nil)
				if err == nil || attempts.Load() != 1 || diagnostics.AttemptCount != 1 || diagnostics.TerminalReason != TerminalReasonSourceUnavailable {
					t.Fatalf("attempts = %d, diagnostics = %+v, err = %v", attempts.Load(), diagnostics, err)
				}
			})

			t.Run("transport exhausts exactly three attempts", func(t *testing.T) {
				var attempts atomic.Int64
				server := groupedServer(validProviderBody(provider))
				defer server.Close()
				diagnostics, err := resolveProvider(t, provider, context.Background(), server, func(universe *Resolver, prior *PriorCloseResolver) {
					client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
						attempts.Add(1)
						return nil, errors.New("synthetic transport failure")
					})}
					universe.HTTPClient = client
					prior.HTTPClient = client
				})
				if err == nil || attempts.Load() != maximumAttempts || diagnostics.AttemptCount != maximumAttempts || diagnostics.RequestCount != 1 {
					t.Fatalf("attempts = %d, diagnostics = %+v, err = %v", attempts.Load(), diagnostics, err)
				}
			})

			t.Run("exact attempt and operation deadlines", func(t *testing.T) {
				var operationDurations []time.Duration
				var attemptDurations []time.Duration
				server := groupedServer(validProviderBody(provider))
				defer server.Close()
				diagnostics, err := resolveProvider(t, provider, context.Background(), server, func(universe *Resolver, prior *PriorCloseResolver) {
					operationFactory := func(parent context.Context, duration time.Duration) (context.Context, context.CancelFunc) {
						operationDurations = append(operationDurations, duration)
						return context.WithCancel(parent)
					}
					attemptFactory := func(parent context.Context, duration time.Duration) (context.Context, context.CancelFunc) {
						attemptDurations = append(attemptDurations, duration)
						return context.WithCancel(parent)
					}
					universe.operationContext, prior.operationContext = operationFactory, operationFactory
					universe.attemptContext, prior.attemptContext = attemptFactory, attemptFactory
				})
				if err != nil || len(operationDurations) != 1 || operationDurations[0] != operationTimeout || len(attemptDurations) != 1 || attemptDurations[0] != requestAttemptTimeout || diagnostics.AttemptCount != 1 {
					t.Fatalf("operation = %v, attempts = %v, diagnostics = %+v, err = %v", operationDurations, attemptDurations, diagnostics, err)
				}
			})

			t.Run("attempt deadlines exhaust exactly three attempts", func(t *testing.T) {
				var durations []time.Duration
				server := groupedServer(validProviderBody(provider))
				defer server.Close()
				diagnostics, err := resolveProvider(t, provider, context.Background(), server, func(universe *Resolver, prior *PriorCloseResolver) {
					factory := func(parent context.Context, duration time.Duration) (context.Context, context.CancelFunc) {
						durations = append(durations, duration)
						return context.WithDeadline(parent, time.Now().Add(-time.Second))
					}
					universe.attemptContext, prior.attemptContext = factory, factory
				})
				if err == nil || len(durations) != maximumAttempts || diagnostics.RequestCount != 1 || diagnostics.AttemptCount != maximumAttempts || diagnostics.TerminalReason != TerminalReasonDeadline {
					t.Fatalf("durations = %v, diagnostics = %+v, err = %v", durations, diagnostics, err)
				}
				for _, duration := range durations {
					if duration != requestAttemptTimeout {
						t.Fatalf("attempt timeout = %v", duration)
					}
				}
			})

			t.Run("operation deadline is terminal before requests", func(t *testing.T) {
				server := groupedServer(validProviderBody(provider))
				defer server.Close()
				diagnostics, err := resolveProvider(t, provider, context.Background(), server, func(universe *Resolver, prior *PriorCloseResolver) {
					factory := func(parent context.Context, duration time.Duration) (context.Context, context.CancelFunc) {
						if duration != operationTimeout {
							t.Fatalf("operation timeout = %v", duration)
						}
						return context.WithDeadline(parent, time.Now().Add(-time.Second))
					}
					universe.operationContext, prior.operationContext = factory, factory
				})
				if err == nil || diagnostics != (AcquisitionDiagnostics{Source: SourceNone, TerminalReason: TerminalReasonDeadline}) {
					t.Fatalf("diagnostics = %+v, err = %v", diagnostics, err)
				}
			})

			t.Run("cancellation is terminal before requests", func(t *testing.T) {
				server := groupedServer(validProviderBody(provider))
				defer server.Close()
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				diagnostics, err := resolveProvider(t, provider, ctx, server, nil)
				if err == nil || diagnostics != (AcquisitionDiagnostics{Source: SourceNone, TerminalReason: TerminalReasonCanceled}) {
					t.Fatalf("diagnostics = %+v, err = %v", diagnostics, err)
				}
			})

			t.Run("configuration failure makes no request", func(t *testing.T) {
				server := groupedServer(validProviderBody(provider))
				defer server.Close()
				diagnostics, err := resolveProvider(t, provider, context.Background(), server, func(universe *Resolver, prior *PriorCloseResolver) {
					universe.APIKey = ""
					prior.APIKey = ""
				})
				if err == nil || diagnostics != (AcquisitionDiagnostics{Source: SourceNone, TerminalReason: TerminalReasonConfiguration}) {
					t.Fatalf("diagnostics = %+v, err = %v", diagnostics, err)
				}
			})
		})
	}
}

func resolveProvider(t *testing.T, provider referenceProvider, ctx context.Context, server *httptest.Server, configure func(*Resolver, *PriorCloseResolver)) (AcquisitionDiagnostics, error) {
	t.Helper()
	schedule := loadSchedule(t)
	facts := scheduleFacts(t, schedule, "2026-07-29")
	universeResolver := testResolver(t, schedule, server)
	priorResolver := testPriorResolver(t, schedule, server)
	if configure != nil {
		configure(universeResolver, priorResolver)
	}
	var err error
	switch provider {
	case universeProvider:
		var result Universe
		result, err = universeResolver.Resolve(ctx, facts)
		if err == nil {
			return result.Diagnostics(), nil
		}
	case priorProvider:
		var result PriorCloses
		result, err = priorResolver.Resolve(ctx, facts, acceptedUniverse(t, facts.TradingDate, []string{"AAA"}))
		if err == nil {
			return result.Diagnostics(), nil
		}
	default:
		t.Fatalf("unknown provider %q", provider)
	}
	return acquisitionErrorDiagnostics(t, err), err
}

func validProviderBody(provider referenceProvider) string {
	if provider == universeProvider {
		return `{"status":"OK","count":1,"results":[{"ticker":"AAA","active":true,"market":"stocks","locale":"us","type":"CS"}]}`
	}
	return `{"status":"OK","adjusted":true,"resultsCount":0,"results":[]}`
}

func assertProviderPageCount(t *testing.T, provider referenceProvider, diagnostics AcquisitionDiagnostics, universePages int) {
	t.Helper()
	want := 0
	if provider == universeProvider {
		want = universePages
	}
	if diagnostics.PageCount != want {
		t.Fatalf("page count = %d, want %d", diagnostics.PageCount, want)
	}
}

func TestUniversePaginationDiagnosticsAreSequentialAndExact(t *testing.T) {
	var server *httptest.Server
	var requests atomic.Int64
	server = httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		request := requests.Add(1)
		if request == 1 {
			_, _ = io.WriteString(writer, `{"status":"OK","count":1,"results":[{"ticker":"AAA","active":true,"market":"stocks","locale":"us","type":"CS"}],"next_url":"`+server.URL+`/v3/reference/tickers?cursor=2"}`)
			return
		}
		_, _ = io.WriteString(writer, `{"status":"OK","count":1,"results":[{"ticker":"BBB","active":true,"market":"stocks","locale":"us","type":"CS"}]}`)
	}))
	defer server.Close()
	diagnostics, err := resolveProvider(t, universeProvider, context.Background(), server, nil)
	if err != nil || diagnostics != (AcquisitionDiagnostics{Source: SourceFresh, RequestCount: 2, PageCount: 2, AttemptCount: 2, TerminalReason: TerminalReasonNone}) {
		t.Fatalf("diagnostics = %+v, err = %v", diagnostics, err)
	}
}
