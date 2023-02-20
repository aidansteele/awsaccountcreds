package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
	"io"
	"time"
)

func getInstanceIdentityConfig(ctx context.Context) (aws.Config, string, error) {
	noCredsCfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return aws.Config{}, "", fmt.Errorf("loading default config: %w", err)
	}

	client := imds.NewFromConfig(noCredsCfg)

	doc, err := client.GetInstanceIdentityDocument(ctx, &imds.GetInstanceIdentityDocumentInput{})
	if err != nil {
		return aws.Config{}, "", fmt.Errorf("getting instance identity doc: %w", err)
	}

	getInstanceCredentials, err := client.GetMetadata(ctx, &imds.GetMetadataInput{Path: "identity-credentials/ec2/security-credentials/ec2-instance"})
	if err != nil {
		return aws.Config{}, "", fmt.Errorf("getting instance identity credentials: %w", err)
	}

	instanceCredsBytes, err := io.ReadAll(getInstanceCredentials.Content)
	if err != nil {
		return aws.Config{}, "", fmt.Errorf("reading instance identity credentials: %w", err)
	}

	instanceCreds := ec2RoleCreds{}
	err = json.Unmarshal(instanceCredsBytes, &instanceCreds)
	if err != nil {
		return aws.Config{}, "", fmt.Errorf("parsing instance identity credentials: %w", err)
	}

	provider := credentials.NewStaticCredentialsProvider(
		instanceCreds.AccessKeyID,
		instanceCreds.SecretAccessKey,
		instanceCreds.Token,
	)

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(doc.Region),
		config.WithCredentialsProvider(provider),
	)
	if err != nil {
		return aws.Config{}, "", fmt.Errorf("loading config: %w", err)
	}

	return cfg, doc.InstanceID, nil
}

// copied from https://github.com/aws/amazon-ssm-agent
// ec2RoleCreds defines the structure for EC2 credentials returned from IMDS
// Copied from github.com/aws/credentials/ec2rolecreds/ec2_role_provider.go
// A ec2RoleCredRespBody provides the shape for unmarshalling credential
// request responses.
type ec2RoleCreds struct {
	// Success State
	Expiration      time.Time
	AccessKeyID     string
	SecretAccessKey string
	Token           string

	// Error state
	Code    string
	Message string
}
