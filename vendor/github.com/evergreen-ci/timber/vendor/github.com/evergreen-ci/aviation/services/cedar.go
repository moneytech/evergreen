package services

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/evergreen-ci/aviation"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

// DialCedarOptions describes the options for the DialCedar function. The base
// address defaults to `cedar.mongodb.com` and the RPC port to 7070. If a base
// address is provided the RPC port must also be provided. The username and
// either password or API key must always be provided.
type DialCedarOptions struct {
	BaseAddress string
	RPCPort     string
	Username    string
	Password    string
	APIKey      string
	Retries     int
}

func (opts *DialCedarOptions) validate() error {
	if opts.Username == "" || (opts.Password == "" && opts.APIKey == "") {
		return errors.New("must provide username and password or API key")
	}

	if opts.BaseAddress == "" {
		opts.BaseAddress = "cedar.mongodb.com"
		opts.RPCPort = "7070"
	}

	if opts.RPCPort == "" {
		return errors.New("must provide the RPC port")
	}

	return nil
}

type userCredentials struct {
	Username string `json:"username"`
	Password string `json:"password,omitempty"`
	apiKey   string
}

const (
	APIUserHeader = "Api-User"
	APIKeyHeader  = "Api-Key"
)

// DialCedar is a convenience function for creating a RPC client connection
// with cedar via gRPC.
func DialCedar(ctx context.Context, client *http.Client, opts *DialCedarOptions) (*grpc.ClientConn, error) {
	if err := opts.validate(); err != nil {
		return nil, errors.Wrap(err, "invalid dial cedar options")
	}

	httpAddress := "https://" + opts.BaseAddress

	creds := &userCredentials{
		Username: opts.Username,
		Password: opts.Password,
		apiKey:   opts.APIKey,
	}

	ca, err := makeCedarCertRequest(ctx, client, http.MethodGet, httpAddress+"/rest/v1/admin/ca", nil)
	if err != nil {
		return nil, errors.Wrap(err, "problem getting cedar root cert")
	}
	crt, err := makeCedarCertRequest(ctx, client, http.MethodPost, httpAddress+"/rest/v1/admin/users/certificate", creds)
	if err != nil {
		return nil, errors.Wrap(err, "problem getting cedar user cert")
	}
	key, err := makeCedarCertRequest(ctx, client, http.MethodPost, httpAddress+"/rest/v1/admin/users/certificate/key", creds)
	if err != nil {
		return nil, errors.Wrap(err, "problem getting cedar user key")
	}

	tlsConf, err := aviation.GetClientTLSConfig(ca, crt, key)
	if err != nil {
		return nil, errors.Wrap(err, "problem creating TLS config")
	}

	return aviation.Dial(ctx, aviation.DialOptions{
		Address: opts.BaseAddress + ":" + opts.RPCPort,
		Retries: opts.Retries,
		TLSConf: tlsConf,
	})
}

func makeCedarCertRequest(ctx context.Context, client *http.Client, method, url string, creds *userCredentials) ([]byte, error) {
	var body io.Reader
	if creds != nil {
		payload, err := json.Marshal(creds)
		if err != nil {
			return nil, errors.Wrap(err, "marshalling credentials payload")
		}
		body = bytes.NewBuffer(payload)
	}
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, errors.Wrap(err, "problem creating http request")
	}
	req = req.WithContext(ctx)

	if creds != nil && creds.Username != "" && creds.apiKey != "" {
		req.Header.Set(APIUserHeader, creds.Username)
		req.Header.Set(APIKeyHeader, creds.apiKey)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "problem with request")
	}
	defer resp.Body.Close()

	out, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "problem reading response")
	}

	if resp.StatusCode != http.StatusOK {
		return out, errors.Errorf("failed request with status code %d", resp.StatusCode)
	}

	return out, nil
}
