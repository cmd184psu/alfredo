package alfredo

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockS3Client struct {
	s3iface.S3API
	mock.Mock
}

func (m *MockS3Client) ListObjectsV2WithContext(ctx context.Context, input *s3.ListObjectsV2Input, opts ...request.Option) (*s3.ListObjectsV2Output, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*s3.ListObjectsV2Output), args.Error(1)
}

func (m *MockS3Client) HeadObject(input *s3.HeadObjectInput) (*s3.HeadObjectOutput, error) {
	args := m.Called(input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*s3.HeadObjectOutput), args.Error(1)
}

func (m *MockS3Client) HeadObjectWithContext(ctx context.Context, input *s3.HeadObjectInput, opts ...request.Option) (*s3.HeadObjectOutput, error) {
	VerbosePrintf("=====Mocking HeadObjectWithContext====")

	args := m.Called(ctx, input)

	if args == nil {
		panic("args is nil")
	}

	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*s3.HeadObjectOutput), args.Error(1)
}

func (m *MockS3Client) GetObjectWithContext(ctx context.Context, input *s3.GetObjectInput, opts ...request.Option) (*s3.GetObjectOutput, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*s3.GetObjectOutput), args.Error(1)
}

func (m *MockS3Client) CreateMultipartUploadWithContext(ctx context.Context, input *s3.CreateMultipartUploadInput, opts ...request.Option) (*s3.CreateMultipartUploadOutput, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*s3.CreateMultipartUploadOutput), args.Error(1)
}

func (m *MockS3Client) UploadPartWithContext(ctx context.Context, input *s3.UploadPartInput, opts ...request.Option) (*s3.UploadPartOutput, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*s3.UploadPartOutput), args.Error(1)
}

func (m *MockS3Client) CompleteMultipartUploadWithContext(ctx context.Context, input *s3.CompleteMultipartUploadInput, opts ...request.Option) (*s3.CompleteMultipartUploadOutput, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*s3.CompleteMultipartUploadOutput), args.Error(1)
}

func (m *MockS3Client) AbortMultipartUploadWithContext(ctx context.Context, input *s3.AbortMultipartUploadInput, opts ...request.Option) (*s3.AbortMultipartUploadOutput, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*s3.AbortMultipartUploadOutput), args.Error(1)
}

func (m *MockS3Client) PutObjectWithContext(ctx context.Context, input *s3.PutObjectInput, opts ...request.Option) (*s3.PutObjectOutput, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*s3.PutObjectOutput), args.Error(1)
}

func TestMigrationMgrStruct_MigrateObject(t *testing.T) {
	srcS3c := &S3ClientSession{Client: &s3.S3{}, Bucket: "source-bucket", ObjectKey: "test-key"}
	tgtS3c := &S3ClientSession{Client: &s3.S3{}, Bucket: "source-bucket", ObjectKey: "test-key"}
	mockSourceS3 := new(MockS3Client)
	mockTargetS3 := new(MockS3Client)
	progress := &ProgressTracker{}
	mgr := NewMigrationManager(srcS3c, tgtS3c, progress, 100)

	srcBody := aws.ReadSeekCloser(strings.NewReader(strings.Repeat("a", 1024)))
	tgtBody := aws.ReadSeekCloser(strings.NewReader(strings.Repeat("a", 1024)))
	size := int64(1024)
	//	srcLM := aws.Time(time.Now().Add(-time.Hour))
	//	tgtLM := aws.Time(time.Now())
	srcLM := aws.Time(time.Now())
	tgtLM := aws.Time(time.Now().Add(-time.Hour))
	mockSourceS3.On("HeadObjectWithContext", mock.Anything, mock.Anything).Return(&s3.HeadObjectOutput{ContentLength: aws.Int64(size), LastModified: srcLM}, nil)
	mockTargetS3.On("HeadObjectWithContext", mock.Anything, mock.Anything).Return(&s3.HeadObjectOutput{ContentLength: aws.Int64(size), LastModified: tgtLM}, nil)
	mockTargetS3.On("PutObjectWithContext", mock.Anything, mock.Anything).Return(&s3.PutObjectOutput{}, nil)

	mockSourceS3.On("GetObjectWithContext", mock.Anything, mock.Anything).Return(&s3.GetObjectOutput{Body: srcBody}, nil)
	mockTargetS3.On("GetObjectWithContext", mock.Anything, mock.Anything).Return(&s3.GetObjectOutput{Body: tgtBody}, nil)

	VerbosePrintf("Mocking successful, running migration")

	err := mgr.MigrateObject(size)
	assert.NoError(t, err)

	mockSourceS3.AssertExpectations(t)
	mockTargetS3.AssertExpectations(t)
}

// func TestMigrationMgrStruct_MigrateObject_Multipart(t *testing.T) {
// 	srcS3c := &S3ClientSession{Client: &s3.S3{}, Bucket: "source-bucket", ObjectKey: "test-key"}
// 	tgtS3c := &S3ClientSession{Client: &s3.S3{}, Bucket: "source-bucket", ObjectKey: "test-key"}

// 	srcS3c.Client = new(MockS3Client)
// 	tgtS3c.Client = new(MockS3Client)
// 	progress := &ProgressTracker{}
// 	mgr := NewMigrationManager(srcS3c, tgtS3c, progress, 100)
// 	size := int64(10 * 1024 * 1024) // 10 MB

// 	srcS3c.Client.On("HeadObjectWithContext", mock.Anything, mock.Anything).Return(&s3.HeadObjectOutput{ContentLength: aws.Int64(size)}, nil)
// 	mockSourceS3.On("GetObjectWithContext", mock.Anything, mock.Anything).Return(&s3.GetObjectOutput{Body: aws.ReadSeekCloser(strings.NewReader(strings.Repeat("a", int(size))))}, nil)
// 	mockTargetS3.On("CreateMultipartUploadWithContext", mock.Anything, mock.Anything).Return(&s3.CreateMultipartUploadOutput{UploadId: aws.String("upload-id")}, nil)
// 	mockTargetS3.On("UploadPartWithContext", mock.Anything, mock.Anything).Return(&s3.UploadPartOutput{ETag: aws.String("etag")}, nil)
// 	mockTargetS3.On("CompleteMultipartUploadWithContext", mock.Anything, mock.Anything).Return(&s3.CompleteMultipartUploadOutput{}, nil)

// 	err := mgr.MigrateObject(size)
// 	assert.NoError(t, err)

//		mockSourceS3.AssertExpectations(t)
//		mockTargetS3.AssertExpectations(t)
//	}
// func TestMigrationMgrStruct_MigrateObject_Error(t *testing.T) {
// 	mockSourceS3 := new(MockS3Client)
// 	mockTargetS3 := new(MockS3Client)
// 	progress := &ProgressTracker{}
// 	mgr := NewMigrationManager(&S3ClientSession{Client: mockSourceS3, Bucket: "source-bucket"}, &S3ClientSession{Client: mockTargetS3, Bucket: "target-bucket"}, progress, 100)

// 	srcKey := "test-key"
// 	tgtKey := "test-key"
// 	size := int64(1024)

// 	mockSourceS3.On("HeadObjectWithContext", mock.Anything, mock.Anything).Return(nil, errors.New("head object error"))

// 	err := mgr.MigrateObject(srcKey, tgtKey, size)
// 	assert.Error(t, err)
// 	assert.Equal(t, "failed to get source object details: head object error", err.Error())

// 	mockSourceS3.AssertExpectations(t)
// 	mockTargetS3.AssertExpectations(t)
// }
