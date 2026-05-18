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
	"github.com/aws/smithy-go/middleware"
	smithyhttp "github.com/aws/smithy-go/transport/http"
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
		config.WithHTTPClient(&http.Client{
			Transport: &debugTransport{wrapped: http.DefaultTransport},
		}))
	if err != nil {
		return nil, err
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = true
		o.APIOptions = append(o.APIOptions, func(stack *middleware.Stack) error {
			insertErr := stack.Finalize.Insert(
				middleware.FinalizeMiddlewareFunc("StripSDKHeaders",
					func(ctx context.Context, in middleware.FinalizeInput, next middleware.FinalizeHandler) (middleware.FinalizeOutput, middleware.Metadata, error) {
						if req, ok := in.Request.(*smithyhttp.Request); ok {
							req.Header.Del("Amz-Sdk-Invocation-Id")
							req.Header.Del("Amz-Sdk-Request")
							req.Header.Del("Accept-Encoding")
						}
						return next.HandleFinalize(ctx, in)
					},
				),
				"Signing",
				middleware.Before,
			)
			// Presign client uses a different stack without "Signing" — safe to ignore
			_ = insertErr
			return nil
		})
	})

	return client, nil
}
