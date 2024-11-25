package alfredo

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"sync/atomic"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// func TestS3ClientSession_ListObjectsWithPrefix(t *testing.T) {
// 	SetVerbose(true)
// 	type fields struct {
// 		Credentials S3credStruct
// 		Bucket      string
// 		Endpoint    string
// 		Region      string
// 		Client      *s3.S3
// 		Versioning  bool
// 		established bool
// 		keepBucket  bool
// 		PolicyId    string
// 	}
// 	type args struct {
// 		prefix string
// 	}
// 	var s3c S3ClientSession
// 	s3c.Load("getobjecttest.json")
// 	if err := s3c.EstablishSession(); err != nil {
// 		t.Error("failed to establish S3 session")
// 		t.Errorf("error=%s\n", err.Error())
// 		os.Exit(1)
// 	}
// 	var wanted []string
// 	//	wanted = append(wanted, "")

// 	tests := []struct {
// 		name    string
// 		fields  fields
// 		args    args
// 		want    []string
// 		wantErr bool
// 	}{
// 		{
// 			name: "base test",
// 			fields: fields{
// 				Credentials: s3c.Credentials,
// 				established: true,
// 				Client:      s3c.Client,
// 			},
// 			args: args{
// 				prefix: "10May2024-tgt/",
// 			},
// 			want: wanted,
// 		},
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			s3c := S3ClientSession{
// 				Credentials: tt.fields.Credentials,
// 				Bucket:      tt.fields.Bucket,
// 				Endpoint:    tt.fields.Endpoint,
// 				Region:      tt.fields.Region,
// 				Client:      tt.fields.Client,
// 				Versioning:  tt.fields.Versioning,
// 				established: tt.fields.established,
// 				keepBucket:  tt.fields.keepBucket,
// 				PolicyId:    tt.fields.PolicyId,
// 			}
// 			got, err := s3c.ListObjectsWithPrefix(tt.args.prefix)
// 			if (err != nil) != tt.wantErr {
// 				t.Errorf("S3ClientSession.ListObjectsWithPrefix() error = %v, wantErr %v", err, tt.wantErr)
// 				return
// 			}
// 			for g := 0; g < len(got); g++ {
// 				VerbosePrintln(got[g])
// 			}

// 			if !reflect.DeepEqual(got, tt.want) {
// 				t.Errorf("S3ClientSession.ListObjectsWithPrefix() = %v, want %v", got, tt.want)
// 			}
// 		})
// 	}
// }

// func TestS3ClientSession_PresignedURL(t *testing.T) {
// 	type fields struct {
// 		Credentials S3credStruct
// 		Bucket      string
// 		Endpoint    string
// 		Region      string
// 		Client      *s3.S3
// 		Versioning  bool
// 		established bool
// 		keepBucket  bool
// 		PolicyId    string
// 		ObjectKey   string
// 	}
// 	type args struct {
// 		objectKey    string
// 		expiredHours int
// 	}
// 	var s3c S3ClientSession
// 	s3c.Load("getobjecttest.json")
// 	if err := s3c.EstablishSession(); err != nil {
// 		t.Error("failed to establish S3 session")
// 		t.Errorf("error=%s\n", err.Error())
// 		os.Exit(1)
// 	}

// 	tests := []struct {
// 		name    string
// 		fields  fields
// 		args    args
// 		want    string
// 		wantErr bool
// 	}{
// 		{
// 			name: "base test",
// 			fields: fields{
// 				Credentials: s3c.Credentials,
// 				Bucket:      "purity-20161205",
// 				Region:      "region",
// 				established: true,
// 				Client:      s3c.Client,
// 				ObjectKey:   "10May2024-tgt/nasunifilere18e74f0-5c04-498a-8f73-90ef5d659d7e-2-0.CLOUDIAN_METADATA.1.log",
// 			},

// 			args: args{
// 				objectKey:    "10May2024-tgt/10May2024-tgt/nasunifilere18e74f0-5c04-498a-8f73-90ef5d659d7e-2-0.CLOUDIAN_METADATA.1.log",
// 				expiredHours: 2,
// 			},
// 		},
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			// s3c := &S3ClientSession{
// 			// 	Credentials: tt.fields.Credentials,
// 			// 	Bucket:      tt.fields.Bucket,
// 			// 	ObjectKey:   tt.fields.ObjectKey,
// 			// 	Endpoint:    tt.fields.Endpoint,
// 			// 	Region:      tt.fields.Region,
// 			// 	Client:      tt.fields.Client,
// 			// 	Versioning:  tt.fields.Versioning,
// 			// 	established: tt.fields.established,
// 			// 	keepBucket:  tt.fields.keepBucket,
// 			// 	PolicyId:    tt.fields.PolicyId,
// 			// }
// 			s3c.Load("getobjecttest.json")
// 			got, err := s3c.PresignedURL(tt.args.expiredHours)
// 			if (err != nil) != tt.wantErr {
// 				t.Errorf("S3ClientSession.PresignedURL() error = %v, wantErr %v", err, tt.wantErr)
// 				return
// 			}
// 			if len(got) == 0 {
// 				t.Errorf("S3ClientSession.PresignedURL() = %v, want %v", got, tt.want)
// 			} else {
// 				fmt.Printf("url=%s\n", got)
// 			}
// 		})
// 	}
// }

// func TestS3ClientSession_GetObjectHash(t *testing.T) {
// 	type fields struct {
// 		Credentials S3credStruct
// 		Bucket      string
// 		Endpoint    string
// 		Region      string
// 		Client      *s3.S3
// 		Versioning  bool
// 		established bool
// 		keepBucket  bool
// 		PolicyId    string
// 		ObjectKey   string
// 	}
// 	var s3c S3ClientSession
// 	s3c.Load("getobjecttest.json")
// 	if err := s3c.EstablishSession(); err != nil {
// 		t.Error("failed to establish S3 session")
// 		t.Errorf("error=%s\n", err.Error())
// 		os.Exit(1)
// 	}

// 	tests := []struct {
// 		name    string
// 		fields  fields
// 		want    string
// 		wantErr bool
// 	}{
// 		{
// 			name: "base test",
// 			fields: fields{
// 				Credentials: s3c.Credentials,
// 				established: true,
// 				Client:      s3c.Client,
// 			},
// 		},
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			s3c := &S3ClientSession{
// 				Credentials: tt.fields.Credentials,
// 				Bucket:      tt.fields.Bucket,
// 				Endpoint:    tt.fields.Endpoint,
// 				Region:      tt.fields.Region,
// 				Client:      tt.fields.Client,
// 				Versioning:  tt.fields.Versioning,
// 				established: tt.fields.established,
// 				keepBucket:  tt.fields.keepBucket,
// 				PolicyId:    tt.fields.PolicyId,
// 				ObjectKey:   tt.fields.ObjectKey,
// 			}
// 			fmt.Printf("key=%s\n", tt.fields.ObjectKey)
// 			fmt.Printf("s3=%s\n", s3c.GetURL())
// 			got, err := s3c.GetObjectHash()
// 			if (err != nil) != tt.wantErr {
// 				t.Errorf("S3ClientSession.GetObjectHash() error = %v, wantErr %v", err, tt.wantErr)
// 				return
// 			}
// 			fmt.Printf("got=%s\n", got)
// 			// if got != tt.want {
// 			// 	t.Errorf("S3ClientSession.GetObjectHash() = %v, want %v", got, tt.want)
// 			// }
// 		})
// 	}
// }

func TestS3ClientSession_ParseFromURL(t *testing.T) {
	type fields struct {
		Credentials S3credStruct
		Bucket      string
		Endpoint    string
		Region      string
		ObjectKey   string
		Client      *s3.S3
		Versioning  bool
		established bool
		keepBucket  bool
		PolicyId    string
	}
	type args struct {
		url string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:   "base test",
			fields: fields{},

			args: args{
				url: "s3://bucket/long/object/key",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s3c := &S3ClientSession{
				Credentials: tt.fields.Credentials,
				Bucket:      tt.fields.Bucket,
				Endpoint:    tt.fields.Endpoint,
				Region:      tt.fields.Region,
				ObjectKey:   tt.fields.ObjectKey,
				Client:      tt.fields.Client,
				Versioning:  tt.fields.Versioning,
				established: tt.fields.established,
				keepBucket:  tt.fields.keepBucket,
				PolicyId:    tt.fields.PolicyId,
			}
			if err := s3c.ParseFromURL(tt.args.url); (err != nil) != tt.wantErr {
				t.Errorf("S3ClientSession.ParseFromURL() error = %v, wantErr %v", err, tt.wantErr)
			} else {
				fmt.Printf("bucket=%q key=%q\n", s3c.Bucket, s3c.ObjectKey)
			}

		})
	}
}

func TestS3ClientSession_RecursiveBucketDeleteAlt(t *testing.T) {
	// type fields struct {
	// 	Credentials S3credStruct
	// 	Bucket      string
	// 	Endpoint    string
	// 	Region      string
	// 	ObjectKey   string
	// 	Client      *s3.S3
	// 	Versioning  bool
	// 	established bool
	// 	keepBucket  bool
	// 	PolicyId    string
	// }
	// tests := []struct {
	// 	name    string
	// 	fields  fields
	// 	wantErr bool
	// }{
	// 	// TODO: Add test cases.
	// }
	// for _, tt := range tests {
	// 	t.Run(tt.name, func(t *testing.T) {
	// 		s3c := S3ClientSession{
	// 			Credentials: tt.fields.Credentials,
	// 			Bucket:      tt.fields.Bucket,
	// 			Endpoint:    tt.fields.Endpoint,
	// 			Region:      tt.fields.Region,
	// 			ObjectKey:   tt.fields.ObjectKey,
	// 			Client:      tt.fields.Client,
	// 			Versioning:  tt.fields.Versioning,
	// 			established: tt.fields.established,
	// 			keepBucket:  tt.fields.keepBucket,
	// 			PolicyId:    tt.fields.PolicyId,
	// 		}
	// 		if err := s3c.RecursiveBucketDeleteAlt(); (err != nil) != tt.wantErr {
	// 			t.Errorf("S3ClientSession.RecursiveBucketDeleteAlt() error = %v, wantErr %v", err, tt.wantErr)
	// 		}
	// 	})
	// }
}

type mockS3Client struct {
	s3iface.S3API
	mock.Mock
}

func (m *mockS3Client) HeadObjectWithContext(ctx aws.Context, input *s3.HeadObjectInput, opts ...request.Option) (*s3.HeadObjectOutput, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*s3.HeadObjectOutput), args.Error(1)
}

func (m *mockS3Client) GetObjectWithContext(ctx aws.Context, input *s3.GetObjectInput, opts ...request.Option) (*s3.GetObjectOutput, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(*s3.GetObjectOutput), args.Error(1)
}

func (m *mockS3Client) PutObjectWithContext(ctx aws.Context, input *s3.PutObjectInput, opts ...request.Option) (*s3.PutObjectOutput, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(*s3.PutObjectOutput), args.Error(1)
}

func (m *mockS3Client) CreateMultipartUploadWithContext(ctx aws.Context, input *s3.CreateMultipartUploadInput, opts ...request.Option) (*s3.CreateMultipartUploadOutput, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(*s3.CreateMultipartUploadOutput), args.Error(1)
}

func (m *mockS3Client) UploadPartWithContext(ctx aws.Context, input *s3.UploadPartInput, opts ...request.Option) (*s3.UploadPartOutput, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(*s3.UploadPartOutput), args.Error(1)
}

func (m *mockS3Client) CompleteMultipartUploadWithContext(ctx aws.Context, input *s3.CompleteMultipartUploadInput, opts ...request.Option) (*s3.CompleteMultipartUploadOutput, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(*s3.CompleteMultipartUploadOutput), args.Error(1)
}

func (m *mockS3Client) AbortMultipartUploadWithContext(ctx aws.Context, input *s3.AbortMultipartUploadInput, opts ...request.Option) (*s3.AbortMultipartUploadOutput, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(*s3.AbortMultipartUploadOutput), args.Error(1)
}

//	func (m *mockS3Client) ListObjectsV2PagesWithContext(ctx context.Context, input *s3.ListObjectsV2Input, fn func(*s3.ListObjectsV2Output, bool) bool) error {
//		args := m.Called(ctx, input, fn)
//		return args.Error(0)
//	}
func (m *mockS3Client) CopyObject(input *s3.CopyObjectInput) (*s3.CopyObjectOutput, error) {
	args := m.Called(input)
	return args.Get(0).(*s3.CopyObjectOutput), args.Error(1)
}
func TestCopyObjectBetweenBuckets(t *testing.T) {
	tests := []struct {
		name          string
		contentLength int64
		shouldError   bool
		errorLocation string
		expectedCalls int
	}{
		{
			name:          "small file success",
			contentLength: defaultPartSizeMin - 1,
			shouldError:   false,
			expectedCalls: 1,
		},
		{
			name:          "small file head object error",
			contentLength: defaultPartSizeMin - 1,
			shouldError:   true,
			errorLocation: "head",
			expectedCalls: 0,
		},
		{
			name:          "large file success",
			contentLength: defaultPartSizeMin + 1024,
			shouldError:   false,
			expectedCalls: 2,
		},
		{
			name:          "Very large file success",
			contentLength: defaultPartSizeMax + 1024,
			shouldError:   false,
			expectedCalls: 2,
		},
		{
			name:          "larger than 5TB should fail",
			contentLength: defaultPartSizeMax*10000 + 1024,
			shouldError:   true,
			expectedCalls: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Initialize mocks
			sourceMock := new(mockS3Client)
			targetMock := new(mockS3Client)
			sourceS3 := S3ClientSession{
				Client: sourceMock, // Type cast using unsafe.Pointer
				Bucket: "source-bucket",
				ctx:    context.Background(),
			}

			targetS3 := S3ClientSession{
				Client: targetMock, // Type cast using unsafe.Pointer
				Bucket: "target-bucket",
				ctx:    context.Background(),
			}

			progress := &ProgressTracker{}

			// Setup mock expectations
			if tt.errorLocation == "head" {
				sourceMock.On("HeadObjectWithContext", mock.Anything, mock.Anything).
					Return(nil, fmt.Errorf("head object error"))
			} else {
				sourceMock.On("HeadObjectWithContext", mock.Anything, mock.Anything).
					Return(&s3.HeadObjectOutput{
						ContentLength: aws.Int64(tt.contentLength),
					}, nil)

				if tt.contentLength < defaultPartSizeMin {
					// Small file expectations
					testData := []byte("test data")
					sourceMock.On("GetObjectWithContext", mock.Anything, mock.Anything).
						Return(&s3.GetObjectOutput{
							Body: io.NopCloser(bytes.NewReader(testData)),
						}, nil)

					targetMock.On("PutObjectWithContext", mock.Anything, mock.Anything).
						Return(&s3.PutObjectOutput{}, nil)
				} else {
					// Large file expectations
					targetMock.On("CreateMultipartUploadWithContext", mock.Anything, mock.Anything).
						Return(&s3.CreateMultipartUploadOutput{
							UploadId: aws.String("test-upload-id"),
						}, nil)

					sourceMock.On("GetObjectWithContext", mock.Anything, mock.Anything).
						Return(&s3.GetObjectOutput{
							Body: io.NopCloser(bytes.NewReader([]byte("test part data"))),
						}, nil)

					targetMock.On("UploadPartWithContext", mock.Anything, mock.Anything).
						Return(&s3.UploadPartOutput{
							ETag: aws.String("test-etag"),
						}, nil)

					targetMock.On("CompleteMultipartUploadWithContext", mock.Anything, mock.Anything).
						Return(&s3.CompleteMultipartUploadOutput{}, nil)
				}
			}

			// Execute test
			err := sourceS3.CopyObjectBetweenBuckets(&targetS3, "source-key", "target-key", progress)

			// Verify results
			if tt.shouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				sourceMock.AssertExpectations(t)
				targetMock.AssertExpectations(t)
			}
		})
	}
}

// MockS3Client is a mock of the S3 client
type MockS3Client struct {
	mock.Mock
}

// func (m *mockS3Client) ListObjectsV2PagesWithContext(ctx context.Context, input *s3.ListObjectsV2Input, fn func(*s3.ListObjectsV2Output, bool) bool) error {
// 	args := m.Called(ctx, input, fn)
// 	return args.Error(0)
// }

func TestCopyAllObjects(t *testing.T) {
	// Create mock S3 clients
	mockSourceS3Client := new(mockS3Client)
	mockTargetS3Client := new(mockS3Client)

	// Create S3ClientSessions
	sourceS3 := S3ClientSession{
		Client: mockSourceS3Client,
		Bucket: "source-bucket",
	}
	targetS3 := &S3ClientSession{
		Client: mockTargetS3Client,
		Bucket: "target-bucket",
	}

	// Create a progress tracker
	progress := &ProgressTracker{}

	// Mock ListObjectsV2PagesWithContext
	mockSourceS3Client.On("ListObjectsV2PagesWithContext", mock.Anything, mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		fn := args.Get(2).(func(*s3.ListObjectsV2Output, bool) bool)
		fn(&s3.ListObjectsV2Output{
			Contents: []*s3.Object{
				{Key: aws.String("object1"), Size: aws.Int64(100)},
				{Key: aws.String("object2"), Size: aws.Int64(200)},
			},
		}, true)
	})

	// Mock CopyObject (assuming it's called in CopyObjectBetweenBuckets)
	mockTargetS3Client.On("CopyObject", mock.Anything).Return(&s3.CopyObjectOutput{}, nil)

	// Call the function
	err := sourceS3.CopyAllObjects(targetS3, progress)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, int64(2), atomic.LoadInt64(&progress.TotalObjects))
	assert.Equal(t, int64(300), atomic.LoadInt64(&progress.TotalBytes))
	assert.Equal(t, int64(2), atomic.LoadInt64(&progress.MigratedObjects))
	assert.Equal(t, int64(0), atomic.LoadInt64(&progress.SkippedObjects))
	assert.Empty(t, progress.FailedObjects)

	// Verify that the mock methods were called as expected
	mockSourceS3Client.AssertExpectations(t)
	mockTargetS3Client.AssertExpectations(t)
}

func TestCopyAllObjectsWithFailure(t *testing.T) {
	// Similar setup as above, but mock a failure scenario
	mockSourceS3Client := new(mockS3Client)
	mockTargetS3Client := new(mockS3Client)

	sourceS3 := S3ClientSession{
		Client: mockSourceS3Client,
		Bucket: "source-bucket",
	}
	targetS3 := &S3ClientSession{
		Client: mockTargetS3Client,
		Bucket: "target-bucket",
	}

	progress := &ProgressTracker{}

	mockSourceS3Client.On("ListObjectsV2PagesWithContext", mock.Anything, mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		fn := args.Get(2).(func(*s3.ListObjectsV2Output, bool) bool)
		fn(&s3.ListObjectsV2Output{
			Contents: []*s3.Object{
				{Key: aws.String("object1"), Size: aws.Int64(100)},
				{Key: aws.String("object2"), Size: aws.Int64(200)},
			},
		}, true)
	})

	// Mock a failure for one of the objects
	mockTargetS3Client.On("CopyObject", mock.MatchedBy(func(input *s3.CopyObjectInput) bool {
		return *input.Key == "object1"
	})).Return(&s3.CopyObjectOutput{}, nil)
	mockTargetS3Client.On("CopyObject", mock.MatchedBy(func(input *s3.CopyObjectInput) bool {
		return *input.Key == "object2"
	})).Return(&s3.CopyObjectOutput{}, errors.New("copy failed"))

	err := sourceS3.CopyAllObjects(targetS3, progress)

	assert.Error(t, err)
	assert.Equal(t, int64(2), atomic.LoadInt64(&progress.TotalObjects))
	assert.Equal(t, int64(300), atomic.LoadInt64(&progress.TotalBytes))
	assert.Equal(t, int64(2), atomic.LoadInt64(&progress.MigratedObjects))
	assert.Equal(t, int64(0), atomic.LoadInt64(&progress.SkippedObjects))
	assert.Len(t, progress.FailedObjects, 1)
	assert.Contains(t, progress.FailedObjects, "object2")

	mockSourceS3Client.AssertExpectations(t)
	mockTargetS3Client.AssertExpectations(t)
}

func TestCalculatePartSize(t *testing.T) {
	// Assuming these constants are defined in your package
	const (
		defaultPartSizeMin int64 = 5 * 1024 * 1024        // 5 MiB
		defaultPartSizeMax int64 = 5 * 1024 * 1024 * 1024 // 5 GiB
		maxParts           int   = 10000
	)

	tests := []struct {
		name     string
		input    int64
		expected int64
	}{
		//desc,
		{"Small file", 1024 * 1024, 0}, // 1 MiB
		{"Just below defaultPartSizeMax", defaultPartSizeMax - 1, defaultPartSizeMin},
		{"Exactly defaultPartSizeMax", defaultPartSizeMax, defaultPartSizeMin},
		{"Large file", 10 * 1024 * 1024 * 1024, defaultPartSizeMin}, // 10 GiB
		{"Very large file", 100 * 1024 * 1024 * 1024, 20971520},     // 100 GiB
		{"Zero size", 0, 0},
		{"Negative size", -1024, 0},
	}
	fmt.Printf("partsize range %s-%s maxparts: %s",
		HumanReadableStorageCapacity(defaultPartSizeMin),
		HumanReadableStorageCapacity(defaultPartSizeMax),
		HumanReadableBigNumber(10000))
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculatePartSize(tt.input)
			if result != tt.expected {
				t.Errorf("CalculatePartSize(%d) = %d; want %d", tt.input, result, tt.expected)
				t.Errorf("(HR)CalculatePartSize(%s) = %s; want %s",
					HumanReadableStorageCapacity(tt.input),
					HumanReadableStorageCapacity(result),
					HumanReadableStorageCapacity(tt.expected))
			}
		})
	}
}

func TestCalculateTotalParts(t *testing.T) {
	tests := []struct {
		name     string
		objSize  int64
		partSize int64
		expected int
	}{
		{"Exact division", 100, 25, 4},
		{"Non-exact division", 100, 30, 4},
		{"Single part", 10, 15, 1},
		{"Zero object size", 0, 10, 0},
		{"Object size less than part size", 5, 10, 1},
		{"Large numbers", 1000000000, 300000000, 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateTotalParts(tt.objSize, tt.partSize)
			if result != tt.expected {
				t.Errorf("CalculateTotalParts(%d, %d) = %d; want %d", tt.objSize, tt.partSize, result, tt.expected)
			}
		})
	}
}
