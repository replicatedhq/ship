package state

import (
	"context"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"testing"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/stretchr/testify/require"
)

func TestURLSerializer(t *testing.T) {
	req := require.New(t)

	srv, err := startStateServer(t)
	req.NoError(err)

	defer srv.Shutdown(context.TODO()) // nolint: errcheck

	s := newURLSerializer(
		log.NewNopLogger(),
		fmt.Sprintf("http://%s/testurlserializer/get-state", srv.Addr),
		fmt.Sprintf("http://%s/testurlserializer/put-state", srv.Addr),
	)

	state1 := State{
		V1: &V1{
			Kustomize: &Kustomize{
				Overlays: map[string]Overlay{
					"ship": {
						Patches: map[string]string{
							"deployment.yaml": "foo/bar/baz",
						},
						Resources: map[string]string{
							"resource.yaml": "hi",
						},
					},
				},
			},
		},
	}

	err = s.Save(state1)
	req.NoError(err)

	state2, err := s.Load()
	req.NoError(err)
	req.Equal(state1, state2)
}

func startStateServer(t *testing.T) (*http.Server, error) {
	req := require.New(t)

	rand.Seed(time.Now().UnixNano())
	port := rand.Intn(32000) + 32000 // random port between 32000 and 64000
	srv := &http.Server{Addr: fmt.Sprintf("127.0.0.1:%d", port)}

	http.HandleFunc("/testurlserializer/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(""))
	})

	var stateData []byte

	http.HandleFunc("/testurlserializer/put-state", func(w http.ResponseWriter, r *http.Request) {
		req.Equal(r.Method, "PUT")
		req.Equal(r.Header.Get("Content-Type"), "application/json")
		body, err := ioutil.ReadAll(r.Body)
		req.NoError(err)
		stateData = body

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(""))
	})

	http.HandleFunc("/testurlserializer/get-state", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(stateData)
	})

	go func() {
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			fmt.Printf("ListenAndServe(): %v", err)
		}
	}()

	var pingErr error
	for i := 0; i < 10; i++ {
		time.Sleep(1 * time.Second)
		resp, err := http.Get(fmt.Sprintf("http://%s/testurlserializer/ping", srv.Addr))
		if err == nil && resp.StatusCode == http.StatusOK {
			return srv, nil
		}
		pingErr = err // safe to assume err is not nil here...
	}

	return nil, pingErr
}
