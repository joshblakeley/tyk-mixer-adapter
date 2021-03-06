// nolint:lll
// Generates the tykgrpcadapter adapter's resource yaml. It contains the adapter's configuration, name,
// supported template names (auth in this case), and whether it is session or no-session based.
//go:generate $REPO_ROOT/bin/mixer_codegen.sh -a mixer/adapter/tykgrpcadapter/config/config.proto -x "-s=false -n tykgrpcadapter -t authorization"

package adapter

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/joshblakeley/tyk-mixer-adapter/pkg/config"
	"google.golang.org/grpc/credentials"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"time"

	"istio.io/istio/mixer/pkg/status"
	"istio.io/istio/mixer/template/authorization"
	"istio.io/pkg/log"

	"google.golang.org/grpc"

	"istio.io/api/mixer/adapter/model/v1beta1"
	policy "istio.io/api/policy/v1beta1"
)

type (

	// Server is basic server interface
	Server interface {
		Addr() string
		Close() error
		Run(shutdown chan error)
	}

	// TykGrpcAdapter supports authorization template.
	TykGRPCAdapter struct {
		listener net.Listener
		server   *grpc.Server
	}

)

// TODO: Utilise analytics template to send analytics record directly to Redis so we get Istio metrics in Tyk Dashboard

var(
	_ authorization.HandleAuthorizationServiceServer = &TykGRPCAdapter{}
 client *http.Client
)


func init() {
	client = createHTTPClient()
}

func createHTTPClient() *http.Client {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	return client
}

// HandleAuthorization handles receiving an auth header from mixer and sending it to a Tyk Gateway for policy validation
// TODO The API Key can be a valid JWT with a corresponding API setup in Tyk
// TODO see: https://tyk.io/docs/basic-config-and-security/security/your-apis/json-web-tokens/
// The key may be a plain bearer token but it needs to have been issued the Tyk Management Dashboard
func (s *TykGRPCAdapter) HandleAuthorization(ctx context.Context, r *authorization.HandleAuthorizationRequest) (*v1beta1.CheckResult, error) {
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
	// TODO: different failure response for different codes i.e. 500/400/404 etc
	value, ok := props["custom_token_header"]
	log.Infof("Header value: %v", value)
	if !ok {
		log.Infof("No authorization header present")
		return &v1beta1.CheckResult{
			Status: status.WithPermissionDenied("Unauthorized..."),
		}, nil
	}

	//send auth key to gateway on the service path
	// TODO: Mutual TLS for connection to Tyk Gateway
	log.Infof("Calling Tyk api on: %s", cfg.GetGatewayUrl()+ "/" + r.Instance.Action.Service + r.Instance.Action.Path)

	req, _ := http.NewRequest("GET",
		cfg.GetGatewayUrl()+"/" + r.Instance.Action.Service + r.Instance.Action.Path,
		nil)
	//TODO: pass custom header in x-tyk-token field with value defined in config
	req.Header.Set("x-tyk-token", value.(string))
	resp, err := client.Do(req)
	if err != nil {
		log.Errorf("error sending request to Tyk gateway: %v", err)
		return &v1beta1.CheckResult{
			Status: status.WithPermissionDenied("Error calling Tyk Gateway"),
		}, nil
	}

	defer resp.Body.Close()
	_,_ = io.Copy(ioutil.Discard, resp.Body)


	log.Infof("StatusCodeFromTyk: %v", resp.StatusCode)

	if resp.StatusCode != 200 {
		return &v1beta1.CheckResult{
			Status: status.WithPermissionDenied("Error Calling Tyk Gateway"),
		}, nil

	}
	return &v1beta1.CheckResult{
		Status: status.OK,
	}, nil
}

// Addr returns the listening address of the server
func (s *TykGRPCAdapter) Addr() string {
	return s.listener.Addr().String()
}

// Run starts the server run
func (s *TykGRPCAdapter) Run(shutdown chan error) {
	shutdown <- s.server.Serve(s.listener)
}

// Close gracefully shuts down the server; used for testing
func (s *TykGRPCAdapter) Close() error {
	if s.server != nil {
		s.server.GracefulStop()
	}

	if s.listener != nil {
		_ = s.listener.Close()
	}

	return nil
}

func getServerTLSOption(credential, privateKey, caCertificate string) (grpc.ServerOption, error) {
	certificate, err := tls.LoadX509KeyPair(
		credential,
		privateKey,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load key cert pair")
	}
	certPool := x509.NewCertPool()
	bs, err := ioutil.ReadFile(caCertificate)
	if err != nil {
		return nil, fmt.Errorf("failed to read client ca cert: %s", err)
	}

	ok := certPool.AppendCertsFromPEM(bs)
	if !ok {
		return nil, fmt.Errorf("failed to append client certs")
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{certificate},
		ClientCAs:    certPool,
	}
	tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert

	return grpc.Creds(credentials.NewTLS(tlsConfig)), nil
}

// TODO: extend config inputs
func NewTykGrpcAdapter(addr string) (*TykGRPCAdapter, error) {

	listener, err := net.Listen("tcp", fmt.Sprintf("%s", ":9999"))
	if err != nil {
		return nil, fmt.Errorf("unable to listen on socket: %v", err)
	}
	s := &TykGRPCAdapter{
		listener: listener,
	}
	fmt.Printf("listening on \"%v\"\n", s.Addr())

	credential := os.Getenv("TYK_GRPC_ADAPTER_CREDENTIAL")
	privateKey := os.Getenv("TYK_GRPC_ADAPTER_PRIVATE_KEY")
	certificate := os.Getenv("TYK_GRPC_ADAPTER_CERTIFICATE")
	if credential != "" {
		so, err := getServerTLSOption(credential, privateKey, certificate)
		if err != nil {
			return nil, err
		}
		s.server = grpc.NewServer(so)
	} else {
		s.server = grpc.NewServer()
	}
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

