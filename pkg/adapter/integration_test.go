package adapter

import (
	"github.com/joshblakeley/tyk-mixer-adapter/pkg/adapter"
	"io/ioutil"
	integration "istio.io/istio/mixer/pkg/adapter/test"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)


func TestAuth(t *testing.T) {
	cases := map[string]struct {
		attrs map[string]interface{}
		want  string
	}{
		"Good api key request": {
			attrs: map[string]interface{}{
				"destination.service.host":"ok",
				"request.headers": map[string]string{"x-tyk-token": "something"},

			},
			want: `
			{
				"AdapterState": null,
				"Returns": [{
					"Check": {
						"Status": {},
						"ValidDuration": 0,
						"ValidUseCount": 1
					},
					"Quota": null,
					"Error": null
				}]
			}
			`,
		},
	}

	ts := httptest.NewServer(TykMockHandler())
	defer ts.Close()

	adptCrBytes, err := ioutil.ReadFile("../config/tykgrpcadapter.yaml")
	if err != nil {
		t.Fatalf("could not read file: %v", err)
	}

	operatorCfgBytes, err := ioutil.ReadFile("../testdata/sample_operator_cfg.yaml")
	if err != nil {
		t.Fatalf("could not read file: %v", err)
	}
	operatorCfg := string(operatorCfgBytes)
	operatorCfg = strings.Replace(operatorCfg,"{TYK_URL}", ts.URL,1)

	shutdown := make(chan error, 1)


	for id, c := range cases {
		t.Logf("** Executing test case '%s' **", id)
		integration.RunTest(
			t,
			nil,
			integration.Scenario{
				ParallelCalls: []integration.Call{
					{
						CallKind: integration.CHECK,
						Attrs:    c.attrs,
					},
				},
				Setup: func() (ctx interface{}, err error) {
					pServer, err := adapter.NewTykGrpcAdapter("")
					if err != nil {
						return nil, err
					}
					go func() {
						pServer.Run(shutdown)
						_ = <-shutdown
					}()
					return pServer, nil
				},
				Teardown: func(ctx interface{}) {
					s := ctx.(adapter.Server)
					s.Close()
				},
				GetState: func(ctx interface{}) (interface{}, error) {
					return nil, nil
				},
				GetConfig: func(ctx interface{}) ([]string, error) {
					s := ctx.(adapter.Server)
					return []string{
						// CRs for built-in templates (authorization is what we need for this test)
						// are automatically added by the integration test framework.
						string(adptCrBytes),
						strings.Replace(operatorCfg, "{ADAPTER_URL}", s.Addr(), 1),
					}, nil
				},

				Want: c.want,
			},
		)
	}
}

func TykMockHandler() http.HandlerFunc {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		switch {

		case strings.HasPrefix(r.URL.Path, "/ok/"):
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`Not found`))

		//case strings.HasPrefix(r.URL.Path, "/ratelimited"):
		//
		//
		//case strings.HasPrefix(r.URL.Path, "/quotalimited"):
		//
		//
		//case strings.HasPrefix(r.URL.Path, "/ok"):
		//	w.Write([]byte(`{"status": "ok"}`))
		//	w.WriteHeader(http.StatusOK)
		}
	})
}

//func normalize(s string) string {
//	s = strings.TrimSpace(s)
//	s = strings.Replace(s, "\t", "", -1)
//	s = strings.Replace(s, "\n", "", -1)
//	s = strings.Replace(s, " ", "", -1)
//	return s
//}
