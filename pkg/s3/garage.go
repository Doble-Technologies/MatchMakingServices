package s3

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type debugTransport struct {
	wrapped http.RoundTripper
}

func (d *debugTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	dump, _ := httputil.DumpRequestOut(req, false) // false = skip body
	fmt.Printf("========== OUTGOING REQUEST ==========\n%s\n", dump)

	resp, err := d.wrapped.RoundTrip(req)

	if resp != nil {
		dumpResp, _ := httputil.DumpResponse(resp, true)
		fmt.Printf("========== RESPONSE ==========\n%s\n", dumpResp)
	}
	return resp, err
}

func NewGarageClient(endpoint, accessKeyID, secretAccessKey, region string) (*s3.Client, error) {
	cfg, err := config.LoadDefaultConfig(
		context.Background(),
		config.WithRegion(region),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, ""),
		),
		config.WithRequestChecksumCalculation(aws.RequestChecksumCalculationWhenRequired),
	)
	if err != nil {
		return nil, err
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = true
	})

	return client, nil
}
