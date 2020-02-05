package adapter

import (
	tyk "github.com/TykTechnologies/tyk/gateway"
	"github.com/TykTechnologies/tyk/user"
	"io/ioutil"
	" github.com/joshblakeley/tyk-mixer-adapter/pkg/adapter"
	adapter_integration "istio.io/istio/mixer/pkg/adapter/test"
	"strings"
	"testing"
)


func TestCheck(t *testing.T) {

	//setup Tyk Server and API
	defer tyk.ResetTestConfig()
	ts := tyk.StartTest()
	defer ts.Close()

	tyk.BuildAndLoadAPI(func(spec *tyk.APISpec) {
		spec.Name = "test"
		spec.APIID = "test"
		spec.Proxy.ListenPath = "/mixerapi/"
		spec.UseKeylessAccess = false
	})
	//create valid auth token and pass to mixer client below
	_, knownKey := ts.CreateSession(func(s *user.SessionState) {
		s.AccessRights = map[string]user.AccessDefinition{"test": {
			APIID: "test",
		}}
	})

	adptCrBytes, err := ioutil.ReadFile("config/tykgrpcadapter.yaml")
	if err != nil {
		t.Fatalf("could not read file: %v", err)
	}

	operatorCfgBytes, err := ioutil.ReadFile("testdata/sample_operator_cfg.yaml")
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
				pServer, err := adapter.NewTykGrpcAdapter("")
				if err != nil {
					return nil, err
				}
				go func() {
					Run(shutdown)
					_ = <-shutdown
				}()
				return pServer, nil
			},
			Teardown: func(ctx interface{}) {
				s := ctx.(adapter.Server)
				Close()
			},
			ParallelCalls: []adapter_integration.Call{
				{
					CallKind: adapter_integration.CHECK,
					Attrs:    map[string]interface{}{
						"request.headers": map[string]string{"x-tyk-token": knownKey},
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
					strings.Replace(operatorCfg, "{ADDRESS}", Addr(), 1),
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






//func normalize(s string) string {
//	s = strings.TrimSpace(s)
//	s = strings.Replace(s, "\t", "", -1)
//	s = strings.Replace(s, "\n", "", -1)
//	s = strings.Replace(s, " ", "", -1)
//	return s
//}
