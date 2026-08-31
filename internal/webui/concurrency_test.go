package webui

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

type blockedResponse struct {
	*httptest.ResponseRecorder
	started chan struct{}
	release chan struct{}
}

func (w *blockedResponse) WriteHeader(status int) {
	close(w.started)
	<-w.release
	w.ResponseRecorder.WriteHeader(status)
}

func TestSlowResponseDoesNotBlockActions(t *testing.T) {
	for _, action := range []bool{false, true} {
		name := "dashboard"
		if action {
			name = "action redirect"
		}
		t.Run(name, func(t *testing.T) {
			testSlowResponse(t, action)
		})
	}
}

func testSlowResponse(t *testing.T, action bool) {
	t.Helper()
	d := newDashboardTest(t, Options{})
	r := httptest.NewRequest(http.MethodGet, "http://localhost:7432/", nil)
	if action {
		fields := url.Values{"csrf": {d.token}, "title": {"First action"}}
		r = httptest.NewRequest(http.MethodPost, "http://localhost:7432/actions/create", strings.NewReader(fields.Encode()))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		r.Header.Set("Origin", "http://localhost:7432")
	}
	w := &blockedResponse{ResponseRecorder: httptest.NewRecorder(), started: make(chan struct{}), release: make(chan struct{})}
	pageDone := make(chan struct{})
	var actionDone chan struct{}
	t.Cleanup(func() {
		close(w.release)
		for _, done := range []chan struct{}{pageDone, actionDone} {
			if done == nil {
				continue
			}
			select {
			case <-done:
			case <-time.After(5 * time.Second):
				t.Error("dashboard request did not finish")
			}
		}
	})
	go func() {
		defer close(pageDone)
		d.handler.ServeHTTP(w, r)
	}()
	select {
	case <-w.started:
	case <-time.After(5 * time.Second):
		t.Fatal("dashboard did not start writing its response")
	}
	actionDone = make(chan struct{})
	result := make(chan *httptest.ResponseRecorder, 1)
	go func() {
		defer close(actionDone)
		result <- d.post("create", url.Values{"title": {"Created while another response is blocked"}})
	}()
	select {
	case res := <-result:
		if res.Code != http.StatusSeeOther {
			t.Fatalf("action returned %d: %s", res.Code, res.Body.String())
		}
	case <-time.After(5 * time.Second):
		t.Fatal("a slow response blocked an unrelated action")
	}
}
