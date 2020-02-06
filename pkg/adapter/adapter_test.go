package adapter

import (
	"context"
	protobuf "github.com/gogo/protobuf/types"
	"github.com/joshblakeley/tyk-mixer-adapter/pkg/adapter"
	"github.com/joshblakeley/tyk-mixer-adapter/pkg/config"
	"io/ioutil"
	"istio.io/api/policy/v1beta1"
	"istio.io/istio/mixer/pkg/status"
	"istio.io/istio/mixer/template/authorization"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"

	"testing"
)

func TestGRPCAdapter_HandleAuthorization(t *testing.T) {
	ctx := context.Background()

	ts := httptest.NewServer(adapter.TykMockHandler())
	defer ts.Close()
	baseURL, err := url.Parse(ts.URL)
	if err != nil {
		t.Fatal(err)
	}

	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("ioutil.TempDir: %s", err)
	}
	defer os.RemoveAll(dir)

	serv, err := adapter.NewTykGrpcAdapter("")
	if err != nil {
		t.Fatalf("unable to start server: %v", err)
	}

	cfg := &config.Params{
		GatewayUrl: baseURL.String(),
	}

	configBytes, err := cfg.Marshal()
	if err != nil {
		t.Fatalf("unable to marshal config: %v", err)
	}
	adapterConfig := &protobuf.Any{
		Value: configBytes,
	}

	instanceMsg := &authorization.InstanceMsg{
		Subject: &authorization.SubjectMsg{
			Properties: map[string]*v1beta1.Value{
				"api_key":     stringValue(""),
				"json_claims": stringValue(""),
			},
		},
		Action: &authorization.ActionMsg{
			Namespace: "default",
			Service:   "service",
			Method:    "GET",
			Path:      "/",
		},
	}

	r := &authorization.HandleAuthorizationRequest{
		Instance:      instanceMsg,
		AdapterConfig: adapterConfig,
	}
	checkResult, err := serv.HandleAuthorization(ctx,r)
	if err != nil {
		t.Errorf("error in HandleAuthorization: %v", err)
	}
	expected := status.WithUnauthenticated("missing authentication")
	if !reflect.DeepEqual(expected, checkResult.Status) {
		t.Errorf("checkResult expected: %v got: %v", expected, checkResult)
	}

	instanceMsg.Subject.Properties["api_key"] = stringValue("badkey")
	checkResult, err = serv.HandleAuthorization(ctx, r)
	if err != nil {
		t.Errorf("error in HandleAuthorization: %v", err)
	}
	expected = status.WithPermissionDenied("permission denied")
	if !reflect.DeepEqual(expected, checkResult.Status) {
		t.Errorf("checkResult expected: %v got: %v", expected, checkResult)
	}

	instanceMsg.Subject.Properties["api_key"] = stringValue("goodkey")
	checkResult, err = serv.HandleAuthorization(ctx, r)
	if err != nil {
		t.Errorf("error in HandleAuthorization: %v", err)
	}
	if !status.IsOK(checkResult.Status) {
		t.Errorf("checkResult expected: %v got: %v", status.OK, checkResult.Status)
	}

	if err := serv.Close(); err != nil {
		t.Errorf("Close() returned an unexpected error")
	}
}

func stringValue(in string) *v1beta1.Value {
	return &v1beta1.Value{
		Value: &v1beta1.Value_StringValue{
			StringValue: in,
		},
	}
}