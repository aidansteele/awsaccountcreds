package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/aidansteele/awsaccountcreds/auth"
	"github.com/aws/aws-sdk-go-v2/aws"
	"io"
	"net/http"
	"time"
)

func requestManagedInstanceRoleToken(ctx context.Context, cfg aws.Config, instanceId string, rsaKey *auth.RsaKey) (*managedInstanceRoleTokenOutput, error) {
	body, _ := json.Marshal(map[string]any{
		"Fingerprint": instanceId,
	})

	ssmEndpoint := fmt.Sprintf("https://ssm.%s.amazonaws.com/", cfg.Region)
	req, _ := http.NewRequestWithContext(ctx, "POST", ssmEndpoint, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "AmazonSSM.RequestManagedInstanceRoleToken")

	v4 := newV4transport(&ssmRsaTransport{
		RoundTripper: http.DefaultTransport,
		key:          rsaKey,
	}, "ssm", cfg)

	c := &http.Client{
		Transport: v4,
		Timeout:   10 * time.Second,
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("making http request: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("non-200: %s", resp.Status)
	}

	defer resp.Body.Close()
	respbody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	output := managedInstanceRoleTokenOutput{}
	err = json.Unmarshal(respbody, &output)
	if err != nil {
		return nil, fmt.Errorf("parsing response body: %w", err)
	}

	return &output, nil
}

type ssmRsaTransport struct {
	http.RoundTripper
	key *auth.RsaKey
}

// copied from https://github.com/aws/amazon-ssm-agent/blob/mainline/agent/ssm/rsaauth/ssm_rsaV4.go
func (s *ssmRsaTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	authZHeader := request.Header.Get("Authorization")
	if len(authZHeader) == 0 {
		return nil, fmt.Errorf("unable to build RSA signature. No Authorization header in request")
	}

	signature, err := s.key.Sign(authZHeader)
	if err != nil {
		return nil, fmt.Errorf("failed to build RSA signature. Err: %v", err)
	}

	request.Header["SSM-AsymmetricKeyAuthorization"] = []string{fmt.Sprintf("Signature=%s", signature)}

	return s.RoundTripper.RoundTrip(request)
}

type managedInstanceRoleTokenOutput struct {
	AccessKeyId         string  `json:"AccessKeyId"`
	SecretAccessKey     string  `json:"SecretAccessKey"`
	SessionToken        string  `json:"SessionToken"`
	TokenExpirationDate float64 `json:"TokenExpirationDate"`
	UpdateKeyPair       bool    `json:"UpdateKeyPair"`
}
