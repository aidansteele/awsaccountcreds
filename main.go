package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

func main() {
	ctx := context.Background()

	// this gets the aws sigv4 credentials from http://169.254.169.254/latest/meta-data/identity-credentials/ec2/security-credentials/ec2-instance
	cfg, instanceId, err := getInstanceIdentityConfig(ctx)
	if err != nil {
		panic(fmt.Sprintf("%+v", err))
	}

	// this either
	//  - loads a previously-saved rsa keypair from disk, or
	//  - generates a new rsa key pair, registers it with SSM, saves it to disk
	// and returns that keypair
	rsaKey, err := getRsaKey(ctx, cfg, instanceId)
	if err != nil {
		panic(fmt.Sprintf("%+v", err))
	}

	// this uses sigv4 *and* RSA-flavoured sigv4 (in SSM-AsymmetricKeyAuthorization header) to
	// request a set of aws creds
	output, err := requestManagedInstanceRoleToken(ctx, cfg, instanceId, rsaKey)
	if err != nil {
		panic(fmt.Sprintf("%+v", err))
	}

	j, _ := json.Marshal(credentialProcessOutput{
		Version:         1,
		AccessKeyId:     output.AccessKeyId,
		SecretAccessKey: output.SecretAccessKey,
		SessionToken:    output.SessionToken,
		Expiration:      time.Unix(int64(output.TokenExpirationDate), 0).Format(time.RFC3339),
	})
	fmt.Println(string(j))
}

type credentialProcessOutput struct {
	Version         int    `json:"Version"`
	AccessKeyId     string `json:"AccessKeyId"`
	SecretAccessKey string `json:"SecretAccessKey"`
	SessionToken    string `json:"SessionToken"`
	Expiration      string `json:"Expiration"`
}
