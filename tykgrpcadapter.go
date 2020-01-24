// nolint:lll
// Generates the tykgrpcadapter adapter's resource yaml. It contains the adapter's configuration, name,
// supported template names (auth in this case), and whether it is session or no-session based.
//go:generate $REPO_ROOT/bin/mixer_codegen.sh -a mixer/adapter/tykgrpcadapter/config/config.proto -x "-s=false -n tykgrpcadapter -t authorization"

package tykgrpcadapter

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"istio.io/istio/mixer/pkg/status"
	"istio.io/istio/mixer/template/authorization"
	"istio.io/pkg/log"

	"google.golang.org/grpc"

	"istio.io/api/mixer/adapter/model/v1beta1"
	policy "istio.io/api/policy/v1beta1"
	"istio.io/istio/mixer/adapter/tykgrpcadapter/config"
)

type (
	// Server is basic server interface
	Server interface {
		Addr() string
		Close() error
		Run(shutdown chan error)
	}

	// TykGrpcAdapter supports authorization template.
	TykGrpcAdapter struct {
		listener net.Listener
		server   *grpc.Server
	}
)

var _ authorization.HandleAuthorizationServiceServer = &TykGrpcAdapter{}

// HandleAuthorization handles receiving an auth header from mixer and sending it to a Tyk Gateway for policy validation
// TODO The API Key can be a valid JWT with a corresponding API setup in Tyk
// TODO see: https://tyk.io/docs/basic-config-and-security/security/your-apis/json-web-tokens/
// The key may be a plain bearer token but it needs to have been issued the Tyk Management Dashboard
func (s *TykGrpcAdapter) HandleAuthorization(ctx context.Context, r *authorization.HandleAuthorizationRequest) (*v1beta1.CheckResult, error) {
	//get tyk gateway url from config
	log.Infof("received request %v\n", *r)

	cfg := &config.Params{}

	if r.AdapterConfig != nil {
		if err := cfg.Unmarshal(r.AdapterConfig.Value); err != nil {
			log.Errorf("error unmarshalling adapter config: %v", err)
			return nil, err
		}
	}

	props := decodeValueMap(r.Instance.Subject.Properties)
	log.Infof("%v", props)

	//dont have header we want so fail
	value, ok := props["custom_token_header"]
	log.Infof("Header value: ", value)
	if !ok {
		log.Infof("No authorization header present")
		return &v1beta1.CheckResult{
			Status: status.WithPermissionDenied("Unauthorized..."),
		}, nil
	}

	//send auth key to gateway
	client := &http.Client{}
	log.Infof("Calling Tyk api on: ", cfg.GetGatewayUrl()+"/mixertestapi/")
	req, _ := http.NewRequest("GET", cfg.GetGatewayUrl()+"/mixertestapi/", nil)
	req.Header.Set("x-tyk-token", value.(string))
	resp, err := client.Do(req)
	if err != nil {
		log.Errorf("error sending request to Tyk gateway", err)
		return &v1beta1.CheckResult{
			Status: status.WithPermissionDenied("Unauthorized..."),
		}, nil
	}
	log.Infof("StatusCodeFromTyk: ", resp.StatusCode)
	//good request send back an ok
	if resp.StatusCode == 200 {
		return &v1beta1.CheckResult{
			Status: status.OK,
		}, nil
	}
	return &v1beta1.CheckResult{
		Status: status.WithPermissionDenied("Unauthorized..."),
	}, nil

}

// Addr returns the listening address of the server
func (s *TykGrpcAdapter) Addr() string {
	return s.listener.Addr().String()
}

// Run starts the server run
func (s *TykGrpcAdapter) Run(shutdown chan error) {
	shutdown <- s.server.Serve(s.listener)
}

// Close gracefully shuts down the server; used for testing
func (s *TykGrpcAdapter) Close() error {
	if s.server != nil {
		s.server.GracefulStop()
	}

	if s.listener != nil {
		_ = s.listener.Close()
	}

	return nil
}

// NewTykGrpcAdapter creates a new IBP adapter that listens at provided port.
func NewTykGrpcAdapter(addr string) (Server, error) {

	listener, err := net.Listen("tcp", fmt.Sprintf("%s", "localhost:5000"))
	if err != nil {
		return nil, fmt.Errorf("unable to listen on socket: %v", err)
	}
	s := &TykGrpcAdapter{
		listener: listener,
	}
	fmt.Printf("listening on \"%v\"\n", s.Addr())
	s.server = grpc.NewServer()
	authorization.RegisterHandleAuthorizationServiceServer(s.server, s)
	return s, nil
}

func decodeValueMap(in map[string]*policy.Value) map[string]interface{} {
	out := make(map[string]interface{}, len(in))
	for k, v := range in {
		out[k] = decodeValue(v.GetValue())
	}
	return out
}

func decodeValue(in interface{}) interface{} {
	switch t := in.(type) {
	case *policy.Value_StringValue:
		return t.StringValue
	case *policy.Value_Int64Value:
		return t.Int64Value
	case *policy.Value_DoubleValue:
		return t.DoubleValue
	default:
		return fmt.Sprintf("%v", in)
	}
}
