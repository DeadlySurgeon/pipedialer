package pipedialer

import (
	"errors"
	"io"
	"net"
	"net/http"
	"sync"
	"testing"
)

var message = "Example"

func TestConnPool(t *testing.T) {
	pool := New()

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(message))
	})

	wg := &sync.WaitGroup{}
	wg.Add(1)

	s := &http.Server{Handler: mux}
	go func() {
		defer wg.Done()
		if err := s.Serve(pool); err != nil && !errors.Is(err, net.ErrClosed) {
			panic(err)
		}
	}()
	defer wg.Wait()
	defer pool.Close()

	trans := http.DefaultTransport.(*http.Transport).Clone()
	trans.DialContext = pool.DialContext
	client := &http.Client{
		Transport: trans,
	}

	resp, err := client.Get("http://resource")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if resp, expected := string(b), message; resp != expected {
		t.Fatal("Expected", expected, "got", `"`+resp+`"`)
	}
}
