package adapter

import (
	"github.com/joshblakeley/tyk-mixer-adapter/pkg/adapter"
	"io/ioutil"
	adapter_integration "istio.io/istio/mixer/pkg/adapter/test"
	"net/http"
	"strings"
	"testing"
)


func TestCheck(t *testing.T) {

	adptCrBytes, err := ioutil.ReadFile("../config/tykgrpcadapter.yaml")
	if err != nil {
		t.Fatalf("could not read file: %v", err)
	}

	operatorCfgBytes, err := ioutil.ReadFile("../testdata/sample_operator_cfg.yaml")
	if err != nil {
		t.Fatalf("could not read file: %v", err)
	}
	operatorCfg := string(operatorCfgBytes)
	shutdown := make(chan error, 1)

	adapter_integration.RunTest(
		t,
		nil,
		adapter_integration.Scenario{
			Setup: func() (ctx interface{}, err error) {
				pServer, err := NewTykGrpcAdapter("")
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
			ParallelCalls: []adapter_integration.Call{
				{
					CallKind: adapter_integration.CHECK,
					Attrs:    map[string]interface{}{
						"request.headers": map[string]string{"x-tyk-token": ""},
						"destination.namespace": "default",
						"destination.service.host": "mixerapi",
						"request.path": "/",
						"request.method": "GET"},
				},
			},
			GetConfig: func(ctx interface{}) ([]string, error) {
				s := ctx.(adapter.Server)
				return []string{
					// CRs for built-in templates (authorization is what we need for this test)
					// are automatically added by the integration test framework.
					string(adptCrBytes),
					strings.Replace(operatorCfg, "{ADDRESS}", s.Addr(), 1),
				}, nil
			},
			Want: `
     {
      "AdapterState": null,
      "Returns": [
       {
        "Check": {
         "Status": {
			"code": 7,
			"message": "h1.handler.istio-system:Unauthorized..."
		}
        },
        "Quota": null,
        "Error": null
       }
      ]
     }`,
		},
	)
}

func tykMockHandler() http.HandlerFunc {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		switch {

		case strings.HasPrefix(r.URL.Path, "/notfound"):
			w.Write([]byte(`Not found`))
			w.WriteHeader(http.StatusNotFound)

		case strings.HasPrefix(r.URL.Path, "/ratelimited"):


		case strings.HasPrefix(r.URL.Path, "/quotalimited"):


		case strings.HasPrefix(r.URL.Path, "/ok"):
			w.Write([]byte(`{"status": "ok"}`))
			w.WriteHeader(http.StatusOK)
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
