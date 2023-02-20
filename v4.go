package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"io"
	"net/http"
	"time"
)

type v4transport struct {
	rt      http.RoundTripper
	service string
	region  string
	creds   *aws.CredentialsCache
}

func newV4transport(rt http.RoundTripper, service string, cfg aws.Config) *v4transport {
	if rt == nil {
		rt = http.DefaultTransport
	}

	return &v4transport{
		rt:      rt,
		service: service,
		region:  cfg.Region,
		creds:   aws.NewCredentialsCache(cfg.Credentials),
	}
}

func (v *v4transport) RoundTrip(request *http.Request) (*http.Response, error) {
	ctx := request.Context()

	var err error
	body := []byte{}

	if request.Body != nil {
		body, err = io.ReadAll(request.Body)
		if err != nil {
			return nil, fmt.Errorf("reading request body: %w", err)
		}

		request.Body = io.NopCloser(bytes.NewReader(body))
	}

	sum := sha256.Sum256(body)
	sumStr := hex.EncodeToString(sum[:])

	creds, err := v.creds.Retrieve(ctx)
	if err != nil {
		return nil, fmt.Errorf("retrieving aws creds: %w", err)
	}

	s := v4.NewSigner()
	err = s.SignHTTP(ctx, creds, request, sumStr, v.service, v.region, time.Now())
	if err != nil {
		return nil, fmt.Errorf("signing request using sigv4: %w", err)
	}

	return v.rt.RoundTrip(request)
}
