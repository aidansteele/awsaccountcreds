package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/aidansteele/awsaccountcreds/auth"
	"github.com/aws/aws-sdk-go-v2/aws"
	"net/http"
	"os"
	"time"
)

const rsaKeyPath = "/tmp/rsakey.txt"

func getRsaKey(ctx context.Context, cfg aws.Config, instanceId string) (*auth.RsaKey, error) {
	rsaKey := loadRsaKey()
	if rsaKey != nil {
		return rsaKey, nil
	}

	return registerNewRsaKey(ctx, cfg, instanceId)
}

func loadRsaKey() *auth.RsaKey {
	keyBytes, err := os.ReadFile(rsaKeyPath)
	if err != nil {
		return nil
	}

	key, err := auth.DecodePrivateKey(string(keyBytes))
	if err != nil {
		return nil
	}

	return &key
}

func registerNewRsaKey(ctx context.Context, cfg aws.Config, instanceId string) (*auth.RsaKey, error) {
	rsaKey, err := auth.CreateKeypair()
	if err != nil {
		return nil, fmt.Errorf("creating rsa key: %w", err)
	}

	err = registerManagedInstance(ctx, cfg, instanceId, rsaKey)
	if err != nil {
		return nil, fmt.Errorf("registering managed instance: %w", err)
	}

	private, err := rsaKey.EncodePrivateKey()
	if err != nil {
		return nil, fmt.Errorf("serializing private key: %w", err)
	}

	err = os.WriteFile(rsaKeyPath, []byte(private), 0600)
	if err != nil {
		return nil, fmt.Errorf("saving private key: %w", err)
	}

	return &rsaKey, nil
}

func registerManagedInstance(ctx context.Context, cfg aws.Config, instanceId string, rsaKey auth.RsaKey) error {
	publicKey, err := rsaKey.EncodePublicKey()
	if err != nil {
		panic(fmt.Sprintf("%+v", err))
	}

	body, _ := json.Marshal(map[string]any{
		"Fingerprint":   instanceId,
		"PublicKey":     publicKey,
		"PublicKeyType": "Rsa",
	})

	v4 := newV4transport(http.DefaultTransport, "ssm", cfg)
	c := &http.Client{
		Transport: v4,
		Timeout:   10 * time.Second,
	}

	ssmEndpoint := fmt.Sprintf("https://ssm.%s.amazonaws.com/", cfg.Region)
	req, _ := http.NewRequestWithContext(ctx, "POST", ssmEndpoint, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "AmazonSSM.RegisterManagedInstance")

	resp, err := c.Do(req)
	if err != nil {
		return fmt.Errorf(": %w", err)
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("non-200: %s", resp.Status)
	}

	return nil
}
