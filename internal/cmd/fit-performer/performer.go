package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"github.com/couchbase/gocbinsights/internal/cmd/fit-performer/cluster"
	"github.com/couchbase/gocbinsights/internal/cmd/fit-performer/fiterror"
	"github.com/couchbase/gocbinsights/internal/cmd/fit-performer/protocol"
	"github.com/couchbase/gocbinsights/internal/cmd/fit-performer/protocol/caps"
	"github.com/couchbase/gocbinsights/internal/cmd/fit-performer/protocol/clustermanagement"
	"github.com/couchbase/gocbinsights/internal/cmd/fit-performer/protocol/metadata"
	"github.com/couchbase/gocbinsights/internal/cmd/fit-performer/protocol/result"
	"github.com/couchbase/gocbinsights/internal/cmd/fit-performer/protocol/shared"
	"github.com/couchbase/gocbinsights/internal/cmd/fit-performer/queryresult"
	"github.com/couchbase/gocbinsights/internal/cmd/fit-performer/unmarshal"
	"google.golang.org/protobuf/types/known/timestamppb"

	cbinsights "github.com/couchbase/gocbinsights"
	"github.com/sirupsen/logrus"
)

type queryHandleResource struct {
	handle        *cbinsights.QueryHandle
	handleTimeout time.Duration
}

type queryStatusResource struct {
	status        *cbinsights.QueryStatus
	handleTimeout time.Duration
}

type resultHandleResource struct {
	handle        *cbinsights.QueryResultHandle
	handleTimeout time.Duration
}

type InsightsPerformer struct {
	conns         *MutexProtectedResources[*cluster.Cluster]
	queries       *MutexProtectedResources[*queryresult.QueryResult]
	queryHandles  *MutexProtectedResources[*queryHandleResource]
	statusHandles *MutexProtectedResources[*queryStatusResource]
	resultHandles *MutexProtectedResources[*resultHandleResource]

	logger *logrus.Logger

	protocol.UnimplementedColumnarCrossServiceServer
	protocol.UnimplementedColumnarServiceServer
}

func NewInsightsPerformer(logger *logrus.Logger) *InsightsPerformer {
	return &InsightsPerformer{
		conns:         NewMutexProtectedResources[*cluster.Cluster](),
		queries:       NewMutexProtectedResources[*queryresult.QueryResult](),
		queryHandles:  NewMutexProtectedResources[*queryHandleResource](),
		statusHandles: NewMutexProtectedResources[*queryStatusResource](),
		resultHandles: NewMutexProtectedResources[*resultHandleResource](),
		logger:        logger,

		UnimplementedColumnarCrossServiceServer: protocol.UnimplementedColumnarCrossServiceServer{},
		UnimplementedColumnarServiceServer:      protocol.UnimplementedColumnarServiceServer{},
	}
}

func (c *InsightsPerformer) FetchPerformerCaps(ctx context.Context, in *caps.FetchPerformerCapsRequest) (*caps.FetchPerformerCapsResponse, error) {
	return &caps.FetchPerformerCapsResponse{
		Sdk:        caps.SDK_SDK_GO,
		SdkVersion: "0.0.1",
		ClusterNewInstance: map[int32]*caps.PerApiElementClusterNewInstance{
			0: {
				SupportsDispatchTimeout: false,
			},
		},
		ClusterClose: map[int32]*caps.PerApiElementClusterClose{
			0: {},
		},
		ClusterExecuteQuery: map[int32]*caps.PerApiElementExecuteQuery{
			0: {
				ExecuteQueryReturns:             caps.PerApiElementExecuteQuery_EXECUTE_QUERY_RETURNS_QUERY_RESULT,
				RowIteration:                    caps.PerApiElementExecuteQuery_ROW_ITERATION_STREAMING_ITERATOR_BASED,
				RowDeserialization:              caps.PerApiElementExecuteQuery_ROW_DESERIALIZATION_STATIC_ROW_TYPING_INDIVIDUAL,
				SupportsPassthroughDeserializer: false,
				SupportsCustomDeserializer:      true,
			},
		},
		ScopeExecuteQuery: map[int32]*caps.PerApiElementExecuteQuery{
			0: {
				ExecuteQueryReturns:             caps.PerApiElementExecuteQuery_EXECUTE_QUERY_RETURNS_QUERY_RESULT,
				RowIteration:                    caps.PerApiElementExecuteQuery_ROW_ITERATION_STREAMING_ITERATOR_BASED,
				RowDeserialization:              caps.PerApiElementExecuteQuery_ROW_DESERIALIZATION_STATIC_ROW_TYPING_INDIVIDUAL,
				SupportsPassthroughDeserializer: false,
				SupportsCustomDeserializer:      true,
			},
		},
		SdkConnectionError: map[int32]*caps.SdkConnectionError{
			0: {
				InvalidCredErrorType: caps.SdkConnectionError_AS_INVALID_CREDENTIAL_EXCEPTION,
				BootstrapErrorType:   caps.SdkConnectionError_AS_COLUMNAR_ERROR,
			},
		},
		AnalyticsProduct:           caps.AnalyticsProduct_ANALYTICS,
		SupportsServerAsyncQueries: true,
		CredentialSupport: &caps.CredentialSupport{
			SupportsJwtCredential:         true,
			SupportsCertificateCredential: true,
			SupportsSetCredential:         true,
		},
	}, nil
}

func (c *InsightsPerformer) Echo(ctx context.Context, in *shared.EchoRequest) (*shared.EchoResponse, error) {
	c.logger.Logf(logrus.InfoLevel, "================ %s : %s ================ ", in.TestName, in.Message)

	return &shared.EchoResponse{}, nil
}

func (c *InsightsPerformer) ClusterNewInstance(ctx context.Context, in *clustermanagement.ClusterNewInstanceRequest) (*result.EmptyResultOrFailureResponse, error) {
	creds, err := parseCredential(in.Credential)
	if err != nil {
		return nil, err
	}

	handleTimeout := cluster.DefaultHandleTimeout

	var timeoutOpts *cbinsights.TimeoutOptions

	if in.Options != nil && in.Options.Timeout != nil {
		timeoutOpts = cbinsights.NewTimeoutOptions()
		if in.Options.Timeout.ConnectTimeout != nil {
			timeoutOpts = timeoutOpts.SetConnectTimeout(in.Options.Timeout.GetConnectTimeout().AsDuration())
		}

		if in.Options.Timeout.QueryTimeout != nil {
			timeoutOpts = timeoutOpts.SetQueryTimeout(in.Options.Timeout.GetQueryTimeout().AsDuration())
		}

		if in.Options.Timeout.HandleTimeout != nil {
			handleTimeout = in.Options.Timeout.GetHandleTimeout().AsDuration()
		}
	}

	var securityOpts *cbinsights.SecurityOptions

	if in.Options != nil && in.Options.Security != nil {
		var trustOnly cbinsights.TrustOnly

		switch {
		case in.Options.Security.GetTrustOnlyCapella():
			trustOnly = cbinsights.TrustOnlyCapella{}
		case in.Options.Security.GetTrustOnlyPlatform():
			trustOnly = cbinsights.TrustOnlySystem{}
		case in.Options.Security.TrustOnlyPemString != nil:
			trustOnly = cbinsights.TrustOnlyPemString{
				Pem: *in.Options.Security.TrustOnlyPemString,
			}
		}

		securityOpts = &cbinsights.SecurityOptions{
			TrustOnly:                            trustOnly,
			DisableServerCertificateVerification: in.Options.Security.DisableServerCertificateVerification,
		}
	}

	var unmarshaler cbinsights.Unmarshaler

	if in.Options != nil && in.Options.Deserializer != nil {
		var err error

		unmarshaler, err = unmarshal.NewUnmarshaler(in.Options.Deserializer)
		if err != nil {
			return nil, err
		}
	}

	var maxRetries *uint32

	if in.Options != nil && in.Options.MaxRetries != nil {
		m := uint32(*in.Options.MaxRetries)
		maxRetries = &m
	}

	logger := cbinsights.NewVerboseLogger()

	start := time.Now()

	sdkCluster, err := cbinsights.NewCluster(in.ConnectionString, creds, &cbinsights.ClusterOptions{
		TimeoutOptions:  timeoutOpts,
		SecurityOptions: securityOpts,
		Unmarshaler:     unmarshaler,
		Logger:          logger,
		MaxRetries:      maxRetries,
	})
	if err != nil {
		return &result.EmptyResultOrFailureResponse{
			Result: fiterror.ConvertSDKErrorToFailureResponse(err),
			Metadata: &metadata.ResponseMetadata{
				ElapsedNanos: time.Since(start).Nanoseconds(),
				Initiated:    timestamppb.New(start),
			},
		}, nil
	}

	c.conns.Set(in.ClusterConnectionId, cluster.New(sdkCluster, handleTimeout))

	return &result.EmptyResultOrFailureResponse{
		Result: &result.EmptyResultOrFailureResponse_EmptySuccess{
			EmptySuccess: true,
		},
		Metadata: &metadata.ResponseMetadata{
			ElapsedNanos: time.Since(start).Nanoseconds(),
			Initiated:    timestamppb.New(start),
		},
	}, nil
}

func (c *InsightsPerformer) SetCredential(ctx context.Context, in *clustermanagement.SetCredentialRequest) (*result.EmptyResultOrFailureResponse, error) {
	cluster, ok := c.conns.Get(in.ExecutionContext.ClusterId)

	if !ok {
		return nil, fmt.Errorf("no cluster found with id %s", in.ExecutionContext.ClusterId)
	}

	start := time.Now()

	creds, err := parseCredential(in.Credential)
	if err != nil {
		return nil, err
	}

	err = cluster.SetCredential(creds)
	if err != nil {
		return &result.EmptyResultOrFailureResponse{
			Result: fiterror.ConvertSDKErrorToFailureResponse(err),
			Metadata: &metadata.ResponseMetadata{
				ElapsedNanos: time.Since(start).Nanoseconds(),
				Initiated:    timestamppb.New(start),
			},
		}, nil
	}

	return &result.EmptyResultOrFailureResponse{
		Result: &result.EmptyResultOrFailureResponse_EmptySuccess{
			EmptySuccess: true,
		},
		Metadata: &metadata.ResponseMetadata{
			ElapsedNanos: time.Since(start).Nanoseconds(),
			Initiated:    timestamppb.New(start),
		},
	}, nil
}

func (c *InsightsPerformer) ClusterClose(ctx context.Context, in *clustermanagement.ClusterCloseRequest) (*result.EmptyResultOrFailureResponse, error) {
	cluster, ok := c.conns.Delete(in.ExecutionContext.ClusterId)

	if !ok {
		return nil, fmt.Errorf("no cluster found with id %s", in.ExecutionContext.ClusterId)
	}

	start := time.Now()

	err := cluster.Close()
	if err != nil {
		return &result.EmptyResultOrFailureResponse{
			Result: fiterror.ConvertSDKErrorToFailureResponse(err),
			Metadata: &metadata.ResponseMetadata{
				ElapsedNanos: time.Since(start).Nanoseconds(),
				Initiated:    timestamppb.New(start),
			},
		}, nil
	}

	return &result.EmptyResultOrFailureResponse{
		Result: &result.EmptyResultOrFailureResponse_EmptySuccess{
			EmptySuccess: true,
		},
		Metadata: &metadata.ResponseMetadata{
			ElapsedNanos: time.Since(start).Nanoseconds(),
			Initiated:    timestamppb.New(start),
		},
	}, nil
}

func (c *InsightsPerformer) CloseAllClusters(ctx context.Context, in *clustermanagement.CloseAllColumnarClustersRequest) (*result.EmptyResultOrFailureResponse, error) {
	clusters := c.conns.Clear()

	start := time.Now()

	for _, cluster := range clusters {
		_ = cluster.Close()
	}

	return &result.EmptyResultOrFailureResponse{
		Result: &result.EmptyResultOrFailureResponse_EmptySuccess{
			EmptySuccess: true,
		},
		Metadata: &metadata.ResponseMetadata{
			ElapsedNanos: time.Since(start).Nanoseconds(),
			Initiated:    timestamppb.New(start),
		},
	}, nil
}

func parseCredential(protoCreds *clustermanagement.ClusterNewInstanceRequest_Credential) (cbinsights.Credential, error) {
	var creds cbinsights.Credential

	switch c := protoCreds.Type.(type) {
	case *clustermanagement.ClusterNewInstanceRequest_Credential_UsernameAndPassword_:
		creds = cbinsights.NewBasicAuthCredential(c.UsernameAndPassword.Username, c.UsernameAndPassword.Password)
	case *clustermanagement.ClusterNewInstanceRequest_Credential_JwtAuth_:
		creds = cbinsights.NewJWTCredential(c.JwtAuth.Jwt)
	case *clustermanagement.ClusterNewInstanceRequest_Credential_CertificateAuth_:
		cert, err := tls.X509KeyPair([]byte(c.CertificateAuth.Cert), []byte(c.CertificateAuth.Key))
		if err != nil {
			return nil, fmt.Errorf("error parsing X509 certificate: %w", err)
		}

		creds = cbinsights.NewCertificateCredential(&cert)
	}

	return creds, nil
}
