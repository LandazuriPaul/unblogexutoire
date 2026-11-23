package main

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"
)

// TestServeDrainsInFlightRequests: on SIGTERM (ctx cancelled) during a rollout, serve must
// not return, and so the process must not exit, before in-flight requests are answered.
func TestServeDrainsInFlightRequests(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	started, release := make(chan struct{}), make(chan struct{})
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-release
		_, _ = w.Write([]byte("fini"))
	})

	ctx, cancel := context.WithCancel(context.Background())
	served := make(chan error, 1)
	go func() { served <- serve(ctx, ln, h) }()

	status := make(chan int, 1)
	go func() {
		resp, err := http.Get("http://" + ln.Addr().String() + "/")
		if err != nil {
			status <- 0
			return
		}
		resp.Body.Close()
		status <- resp.StatusCode
	}()

	<-started
	cancel()
	select {
	case err := <-served:
		t.Fatalf("serve returned (%v) while a request was still in flight", err)
	case <-time.After(200 * time.Millisecond):
	}

	close(release)
	if got := <-status; got != 200 {
		t.Errorf("in-flight request got status %d, want 200", got)
	}
	select {
	case err := <-served:
		if err != nil {
			t.Errorf("serve = %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("serve did not return after the request finished")
	}
}
