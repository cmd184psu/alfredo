package alfredo

import (
	"bytes"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
)

// MPUUpload represents a pending or in-progress multipart upload
type MPUUpload struct {
	Key       string
	UploadID  string
	Initiated time.Time
}

// ListMultipartUploads returns all pending or in-progress multipart uploads for a bucket
func (s3c S3ClientSession) ListMultipartUploads() ([]MPUUpload, error) {
	if err := s3c.EstablishSession(); err != nil {
		return []MPUUpload{}, err
	}
	if len(s3c.Bucket) == 0 {
		return []MPUUpload{}, errors.New("bucket is not set")
	}

	input := &s3.ListMultipartUploadsInput{
		Bucket: aws.String(s3c.Bucket),
	}

	var uploads []MPUUpload

	err := s3c.Client.ListMultipartUploadsPages(input,
		func(page *s3.ListMultipartUploadsOutput, lastPage bool) bool {
			for _, u := range page.Uploads {
				uploads = append(uploads, MPUUpload{
					Key:       aws.StringValue(u.Key),
					UploadID:  aws.StringValue(u.UploadId),
					Initiated: aws.TimeValue(u.Initiated),
				})
			}
			return !lastPage
		})
	if err != nil {
		return nil, fmt.Errorf("failed to list multipart uploads: %w", err)
	}

	return uploads, nil
}

// AbortMultipartUploads aborts all multipart uploads older than cutoff
func (s3c S3ClientSession) AbortMultipartUploads(uploads []MPUUpload, cutoff time.Duration) error {
	VerbosePrintln("BEGIN S3ClientSession::AbortMultipartUploads()")
	defer VerbosePrintln("END S3ClientSession::AbortMultipartUploads()")
	VerbosePrintf("Aborting MPUs older than %s\n", cutoff)
	if err := s3c.EstablishSession(); err != nil {
		return err
	}
	if len(s3c.Bucket) == 0 {
		return errors.New("bucket is not set")
	}

	now := time.Now()

	for _, u := range uploads {
		age := now.Sub(u.Initiated)
		if age < cutoff {
			fmt.Printf("Skipping MPU (too new): Key=%s, UploadID=%s, Age=%s\n", u.Key, u.UploadID, age)
			continue
		}
		fmt.Printf("Aborting MPU: Key=%s, UploadID=%s, Age=%s\n", u.Key, u.UploadID, age)

		_, err := s3c.Client.AbortMultipartUpload(&s3.AbortMultipartUploadInput{
			Bucket:   aws.String(s3c.Bucket),
			Key:      aws.String(u.Key),
			UploadId: aws.String(u.UploadID),
		})
		if err != nil {
			VerbosePrintf("Failed to abort upload %s (%s): %s\n", u.Key, u.UploadID, err)
			return fmt.Errorf("failed to abort upload %s (%s): %w", u.Key, u.UploadID, err)
		}
		fmt.Printf("Aborted MPU: Key=%s, UploadID=%s, Age=%s\n", u.Key, u.UploadID, age)
	}
	VerbosePrintln("Return clean")
	return nil
}

// func main() {
// 	bucket := "your-bucket-name"
// 	region := "us-east-1"

// 	// List uploads
// 	uploads, err := ListMultipartUploads(bucket, region)
// 	if err != nil {
// 		log.Fatalf("Error listing MPUs: %v", err)
// 	}

// 	if len(uploads) == 0 {
// 		fmt.Println("No pending or in-progress multipart uploads.")
// 		return
// 	}

// 	fmt.Printf("Found %d pending/in-progress MPUs:\n", len(uploads))
// 	for _, u := range uploads {
// 		fmt.Printf("Key=%s, UploadID=%s, Initiated=%s\n", u.Key, u.UploadID, u.Initiated)
// 	}

// 	// Example: abort MPUs older than 24 hours
// 	cutoff := 24 * time.Hour
// 	if err := AbortMultipartUploads(bucket, region, uploads, cutoff); err != nil {
// 		log.Fatalf("Error aborting MPUs: %v", err)
// 	}
// }

// CreateIncompleteMultipartUpload starts a multipart upload and uploads a few parts from memory.
// It intentionally leaves the upload incomplete so it can be listed and aborted later.
func (s3c S3ClientSession) CreateIncompleteMultipartUpload(key string, numParts int, partSize int64) (string, error) {
	if err := s3c.EstablishSession(); err != nil {
		return "", err
	}
	if len(s3c.Bucket) == 0 {
		return "", errors.New("bucket is not set")
	}

	// Step 1: initiate MPU
	createResp, err := s3c.Client.CreateMultipartUpload(&s3.CreateMultipartUploadInput{
		Bucket: aws.String(s3c.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return "", fmt.Errorf("failed to initiate MPU: %w", err)
	}

	uploadID := aws.StringValue(createResp.UploadId)
	fmt.Printf("Started MPU: Key=%s, UploadID=%s\n", key, uploadID)

	// Step 2: upload some parts from memory
	for i := 1; i <= numParts; i++ {
		// generate random bytes for this part
		data := make([]byte, partSize)
		rand.Read(data) // fill with random data

		_, err := s3c.Client.UploadPart(&s3.UploadPartInput{
			Bucket:     aws.String(s3c.Bucket),
			Key:        aws.String(key),
			PartNumber: aws.Int64(int64(i)),
			UploadId:   aws.String(uploadID),
			Body:       bytes.NewReader(data),
		})
		if err != nil {
			return uploadID, fmt.Errorf("failed to upload part %d: %w", i, err)
		}
		fmt.Printf("Uploaded part %d (%d bytes)\n", i, partSize)
	}

	// Step 3: leave the MPU incomplete (do not complete it)
	fmt.Printf("Leaving MPU incomplete: Key=%s, UploadID=%s\n", key, uploadID)

	return uploadID, nil
}
