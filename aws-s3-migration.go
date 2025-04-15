package alfredo

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
)

type MigrationMgrStruct struct {
	SourceS3   *S3ClientSession
	TargetS3   *S3ClientSession
	Progress   *ProgressTracker
	ErrorMsg   string
	SourceHead *s3.HeadObjectOutput
	TargetHead *s3.HeadObjectOutput
	output     *s3.ListObjectsV2Output
	WorkerPool chan struct{}
}

func (mgr *MigrationMgrStruct) Lock() {
	mgr.Progress.Lock()
}
func (mgr *MigrationMgrStruct) Unlock() {
	mgr.Progress.Unlock()
}

func (mgr *MigrationMgrStruct) DeepCopy() *MigrationMgrStruct {
	var newMgr MigrationMgrStruct
	s := mgr.SourceS3.DeepCopy()
	t := mgr.TargetS3.DeepCopy()
	newMgr.SourceS3 = &s
	newMgr.TargetS3 = &t
	//still want the same progress pointer
	newMgr.Progress = mgr.Progress
	newMgr.ErrorMsg = ""
	newMgr.SourceHead = nil
	newMgr.TargetHead = nil
	newMgr.output = nil
	newMgr.WorkerPool = make(chan struct{}, mgr.SourceS3.GetConcurrency())
	return &newMgr
}

func NewMigrationManager(sourceS3 *S3ClientSession, targetS3 *S3ClientSession, progress *ProgressTracker, batchSize int) *MigrationMgrStruct {
	// Initialize pagination token
	if batchSize < 1 || batchSize > 1000 {
		batchSize = 1000
	}
	sourceS3.BatchSize = batchSize
	sourceS3.ctx = context.Background()
	targetS3.ctx = context.Background()

	progress.FailedObjects = make(map[string]error)

	return &MigrationMgrStruct{
		SourceS3:   sourceS3,
		TargetS3:   targetS3,
		Progress:   progress,
		WorkerPool: make(chan struct{}, sourceS3.GetConcurrency()),
	}
}

func (mgr *MigrationMgrStruct) MigrationLoop(wg *sync.WaitGroup, ResultsChan *chan CopyResult) error {
	// Create input with pagination token
	input := &s3.ListObjectsV2Input{
		Bucket:            aws.String(mgr.SourceS3.Bucket),
		MaxKeys:           aws.Int64(int64(mgr.SourceS3.BatchSize)), // Process in batches, capped at 1000
		ContinuationToken: mgr.SourceS3.ContinuationToken,
	}
	// Get one page of results
	var err error
	mgr.output, err = mgr.SourceS3.Client.ListObjectsV2WithContext(mgr.SourceS3.ctx, input)
	if err != nil {
		return fmt.Errorf("failed to list objects: %v", err)
	}

	log.Printf("found %d objects in bucket %s, in this page\n", len(mgr.output.Contents), mgr.SourceS3.Bucket)

	if mgr.output.ContinuationToken != nil {
		log.Printf("NextContinuationToken: %s\n", *mgr.output.ContinuationToken)
	} else {
		log.Printf("NextContinuationToken is nil\n")
		panic("")
	}

	for _, obj := range mgr.output.Contents {
		atomic.AddInt64(&mgr.Progress.TotalObjects, 1)
		atomic.AddInt64(&mgr.Progress.TotalBytes, *obj.Size)

		key := *obj.Key
		size := *obj.Size

		mgr.Lock()
		newMgr := mgr.DeepCopy()
		mgr.Unlock()
		newMgr.SourceS3.ObjectKey = key
		newMgr.TargetS3.ObjectKey = key
		wg.Add(1)
		go func(innerMgr *MigrationMgrStruct, objectSize int64) {
			defer wg.Done()

			innerMgr.WorkerPool <- struct{}{}
			defer func() { <-mgr.WorkerPool }()

			startTime := time.Now()
			// err := sourceS3.CopyObjectBetweenBuckets(
			// 	mgr.TargetS3,
			// 	objectKey,
			// 	objectKey,
			// 	mgr.Progress,
			// )
			err := innerMgr.MigrateObject(objectSize)

			result := CopyResult{
				SourceKey:   innerMgr.SourceS3.ObjectKey,
				TargetKey:   innerMgr.TargetS3.ObjectKey,
				Success:     err == nil,
				Error:       err,
				Duration:    time.Since(startTime),
				BytesCopied: objectSize,
			}

			if result.Success {
				log.Printf("Uploaded object to s3://%s/%s", innerMgr.TargetS3.Bucket, result.SourceKey)
			} else if strings.Contains(result.Error.Error(), "skip limit exceeded") {
				log.Printf("Failed to upload object to s3://%s/%s: due to skip size limit exceeded",
					innerMgr.TargetS3.Bucket, result.SourceKey)
			} else {
				log.Printf("Failed to upload object to s3://%s/%s: %v",
					innerMgr.TargetS3.Bucket, result.SourceKey, result.Error)
			}

			select {
			case *ResultsChan <- result:
			case <-time.After(5 * time.Second):
				log.Printf("Warning: Timed out sending result for %s", innerMgr.SourceS3.ObjectKey)
			}
		}(newMgr, size)
	}

	if !mgr.IsDone() {
		log.Printf("Continuing to next page of objects in bucket %s\n", mgr.SourceS3.Bucket)
		if mgr.output.NextContinuationToken != nil {
			mgr.SourceS3.ContinuationToken = mgr.output.NextContinuationToken
		} else {
			log.Printf("NextContinuationToken is nil, stopping pagination\n")
			panic("NextContinuationToken is nil, stopping pagination")
		}
	}

	return nil
}

// type objStruct struct {
// 	ObjectName string
// 	Size       int64
// }

func ParseKeyList(key string) (string, int64) {
	// key := "key1|100.0"
	name := strings.Split(key, "|")[0]
	sizeFloat, err := strconv.ParseFloat(strings.Split(key, "|")[1], 64)
	if err != nil {
		log.Fatalf("failed to parse size: %v", err)
	}
	return name, int64(sizeFloat)
}
func (mgr *MigrationMgrStruct) MigrationBatch(keys []string, wg *sync.WaitGroup, ResultsChan *chan CopyResult) error {
	// Get one page of results
	log.Printf("found %d objects in bucket %s, for this batch\n", len(keys), mgr.SourceS3.Bucket)

	for i := 0; i < len(keys); i++ {
		key, size := ParseKeyList(keys[i])

		atomic.AddInt64(&mgr.Progress.TotalObjects, 1)
		atomic.AddInt64(&mgr.Progress.TotalBytes, size)

		mgr.Lock()
		newMgr := mgr.DeepCopy()
		mgr.Unlock()
		newMgr.SourceS3.ObjectKey = key
		newMgr.TargetS3.ObjectKey = key

		wg.Add(1)
		go func(innerMgr *MigrationMgrStruct, objectSize int64) {
			defer wg.Done()

			mgr.WorkerPool <- struct{}{}
			defer func() { <-mgr.WorkerPool }()

			startTime := time.Now()
			// err := sourceS3.CopyObjectBetweenBuckets(
			// 	mgr.TargetS3,
			// 	objectKey,
			// 	objectKey,
			// 	mgr.Progress,
			// )
			log.Printf("bytes processed: %d / %d objects processed: %d / %d", mgr.Progress.CompletedBytes, mgr.Progress.TotalBytes, mgr.Progress.MigratedObjects, mgr.Progress.TotalObjects)

			err := innerMgr.MigrateObject(objectSize)

			result := CopyResult{
				SourceKey:   innerMgr.SourceS3.ObjectKey,
				TargetKey:   innerMgr.TargetS3.ObjectKey,
				Success:     err == nil,
				Error:       err,
				Duration:    time.Since(startTime),
				BytesCopied: objectSize,
			}

			if result.Success {
				log.Printf("Uploaded object to s3://%s/%s", innerMgr.TargetS3.Bucket, result.SourceKey)
				log.Printf("bytes processed: %d / %d objects processed: %d / %d", mgr.Progress.CompletedBytes, mgr.Progress.TotalBytes, mgr.Progress.MigratedObjects, mgr.Progress.TotalObjects)
			} else if strings.Contains(result.Error.Error(), "skip limit exceeded") {
				log.Printf("Failed to upload object to s3://%s/%s: due to skip size limit exceeded",
					innerMgr.TargetS3.Bucket, result.SourceKey)
			} else {
				log.Printf("Failed to upload object to s3://%s/%s: %v",
					innerMgr.TargetS3.Bucket, result.SourceKey, result.Error)
			}

			select {
			case *ResultsChan <- result:
			case <-time.After(5 * time.Second):
				log.Printf("Warning: Timed out sending result for %s", innerMgr.SourceS3.ObjectKey)
			}
		}(newMgr, size)
	}
	log.Println("Waiting for this batch to complete")
	wg.Wait()
	log.Println("This batch to complete")

	return nil
}

func (mgr *MigrationMgrStruct) CopyObjectBetweenBucketsMPU() error {
	VerbosePrintf("BEGIN CopyObjectBetweenBucketsMPU(%s, %s)\n", mgr.SourceS3.ObjectKey, mgr.TargetS3.ObjectKey)
	if len(mgr.TargetS3.ObjectKey) == 0 {
		mgr.TargetS3.ObjectKey = mgr.SourceS3.ObjectKey
	}
	// Get source object details
	// For large files, use multipart upload with streaming
	log.Printf("Creating MPU for s3://%s/%s", mgr.TargetS3.Bucket, mgr.TargetS3.ObjectKey)
	createOutput, err := mgr.TargetS3.Client.CreateMultipartUploadWithContext(mgr.TargetS3.ctx, &s3.CreateMultipartUploadInput{
		Bucket: aws.String(mgr.TargetS3.Bucket),
		Key:    aws.String(mgr.TargetS3.ObjectKey),
	})
	if err != nil {
		return fmt.Errorf("failed to create multipart upload: %v", err)
	}

	// partSize := CalculatePartSize(*headOutputSrc.ContentLength)
	if mgr.SourceHead.ContentLength == nil {
		return fmt.Errorf("source object does not have a content length")
	}

	objectSize := *mgr.SourceHead.ContentLength
	var partsCount int64
	if mgr.SourceHead.PartsCount == nil {
		log.Printf("!! Source object (s3://%s/%s) missing partsCount in header", mgr.SourceS3.Bucket, mgr.SourceS3.ObjectKey)
		partsCount = CalculateTotalParts(objectSize, CalculatePartSize(objectSize))
	} else {
		partsCount = *mgr.SourceHead.PartsCount
	}
	partSize := (objectSize + partsCount - 1) / partsCount
	partSize = ((partSize + 1048575) / 1048576) * 1048576 // Round up to nearest MB

	//partSize := *log.Printf("Using partsize: %s", HumanReadableStorageCapacity(partSize))
	//totalParts := CalculateTotalParts(*headOutputSrc.ContentLength, partSize)
	//*headOutput.ContentLength + partSize - 1) / partSize
	log.Printf("Using total parts: %d", partsCount)
	log.Printf("Maximum parts: %d", maxParts)
	if partsCount > maxParts {
		panic("requested too many parts for this object")
	}
	log.Printf("Part size range: %s-%s", HumanReadableStorageCapacity(defaultPartSizeMin), HumanReadableStorageCapacity(defaultPartSizeMax))
	parts := make([]*s3.CompletedPart, partsCount)
	partsChan := make(chan int64, partsCount+1)
	errorsChan := make(chan error, partsCount+1)
	var uploadWg sync.WaitGroup

	// Fill parts channel
	for i := int64(1); i <= int64(partsCount); i++ {
		partsChan <- i
	}
	close(partsChan)

	// Process parts concurrently
	for i := 0; i < mgr.SourceS3.GetConcurrency(); i++ {
		uploadWg.Add(1)
		go func() {
			defer uploadWg.Done()

			for partNumber := range partsChan {
				startByte := (partNumber - 1) * partSize
				endByte := startByte + partSize - 1
				if endByte >= objectSize {
					endByte = objectSize - 1
				}

				// Get the part from source
				getPartOutput, err := mgr.SourceS3.Client.GetObjectWithContext(mgr.SourceS3.ctx, &s3.GetObjectInput{
					Bucket: aws.String(mgr.SourceS3.Bucket),
					Key:    aws.String(mgr.SourceS3.ObjectKey),
					Range:  aws.String(fmt.Sprintf("bytes=%d-%d", startByte, endByte)),
				})
				if err != nil {
					errorsChan <- fmt.Errorf("failed to get part %d: %v", partNumber, err)
					return
				}
				body, err := io.ReadAll(getPartOutput.Body)
				if err != nil {
					return
				}
				defer getPartOutput.Body.Close()

				// Create an io.ReadSeeker from the byte slice
				readSeeker := bytes.NewReader(body)

				// Upload the part
				log.Printf("Uploading part of MPU for s3://%s/%s part #: %d of %d", mgr.TargetS3.Bucket, mgr.TargetS3.ObjectKey, partNumber, partsCount)

				uploadOutput, err := mgr.TargetS3.Client.UploadPartWithContext(mgr.TargetS3.ctx, &s3.UploadPartInput{
					Bucket:     aws.String(mgr.TargetS3.Bucket),
					Key:        aws.String(mgr.TargetS3.ObjectKey),
					PartNumber: aws.Int64(partNumber),
					UploadId:   createOutput.UploadId,
					Body:       readSeeker,
				})
				getPartOutput.Body.Close()

				if err != nil {
					log.Printf("Failed to upload part %d: %v", partNumber, err)
					errorsChan <- fmt.Errorf("failed to upload part %d: %v", partNumber, err)
					return
				}
				//log.Printf("using etag: %s", *uploadOutput.ETag)
				parts[partNumber-1] = &s3.CompletedPart{
					ETag:       uploadOutput.ETag,
					PartNumber: aws.Int64(partNumber),
				}

				atomic.AddInt64(&mgr.Progress.CompletedBytes, endByte-startByte+1)
			}
		}()
	}
	uploadWg.Wait()
	close(errorsChan)

	hasErrors := false

	// Check for errors
	for err := range errorsChan {
		// Abort multipart upload
		log.Printf("Aborting MPU for s3://%s/%s due to error: %s", mgr.TargetS3.Bucket, mgr.TargetS3.ObjectKey, err.Error())

		_, abortErr := mgr.TargetS3.Client.AbortMultipartUploadWithContext(mgr.TargetS3.ctx, &s3.AbortMultipartUploadInput{
			Bucket:   aws.String(mgr.TargetS3.Bucket),
			Key:      aws.String(mgr.TargetS3.ObjectKey),
			UploadId: createOutput.UploadId,
		})
		if abortErr != nil {
			log.Fatalf("failed to abort multipart upload: %v (original error: %v)", abortErr, err)
			return fmt.Errorf("failed to abort multipart upload: %v (original error: %v)", abortErr, err)
		}
		hasErrors = true

	}

	if hasErrors {
		log.Fatal("errors occurred; some objects MPU were aborted as a result")
		return fmt.Errorf("errors occurred; some objects MPU were aborted as a result")
	}
	log.Printf("Completed %d parts, checking for nil etags", len(parts))
	for i := 0; i < len(parts); i++ {
		if parts[i] == nil {
			log.Printf("part %d is nil", i)
		} else {
			//log.Printf("part %d is not nil", i)
			// if *parts[i].PartNumber != int64(i) {
			// 	log.Printf("part %d has part number %d", i, parts[i].PartNumber)
			// }
			if parts[i].ETag == nil {
				log.Printf("part %d has nil etag", i)
			}
		}
	}

	// Complete multipart upload
	_, err = mgr.TargetS3.Client.CompleteMultipartUploadWithContext(mgr.TargetS3.ctx, &s3.CompleteMultipartUploadInput{
		Bucket:   aws.String(mgr.TargetS3.Bucket),
		Key:      aws.String(mgr.TargetS3.ObjectKey),
		UploadId: createOutput.UploadId,
		MultipartUpload: &s3.CompletedMultipartUpload{
			Parts: parts,
		},
	})
	if err != nil {
		log.Printf("MPU for s3://%s/%s => s3://%s/%s  failed to complete", mgr.SourceS3.Bucket, mgr.SourceS3.ObjectKey, mgr.TargetS3.Bucket, mgr.TargetS3.ObjectKey)
		return fmt.Errorf("failed to complete multipart upload: %v", err)
	}
	log.Printf("Completing MPU for s3://%s/%s", mgr.TargetS3.Bucket, mgr.TargetS3.ObjectKey)
	log.Printf("Completed %d objects", mgr.Progress.MigratedObjects)

	VerbosePrintf("END CopyObjectBetweenBucketsMPU(%s, %s)\n", mgr.SourceS3.ObjectKey, mgr.TargetS3.ObjectKey)

	return nil
}

func (mgr *MigrationMgrStruct) CopyObjectBetweenBucketsRegular() error {
	tgtKey := mgr.TargetS3.ObjectKey
	if len(mgr.TargetS3.ObjectKey) == 0 {
		mgr.TargetS3.ObjectKey = mgr.SourceS3.ObjectKey
	}
	// Get the object
	getOutput, err := mgr.SourceS3.Client.GetObjectWithContext(mgr.SourceS3.ctx, &s3.GetObjectInput{
		Bucket: aws.String(mgr.SourceS3.Bucket),
		Key:    aws.String(mgr.SourceS3.ObjectKey),
	})
	if err != nil {
		return fmt.Errorf("failed to get source object: %v", err)
	}
	defer getOutput.Body.Close()

	body, err := io.ReadAll(getOutput.Body)
	if err != nil {
		fmt.Printf("Error reading body, err=%s\n", err.Error())
		return err
	}
	defer getOutput.Body.Close()

	// Create an io.ReadSeeker from the byte slice
	readSeeker := bytes.NewReader(body)

	// Now you can use readSeeker where io.ReadSeeker is expected

	// Put the object
	_, err = mgr.TargetS3.Client.PutObjectWithContext(mgr.TargetS3.ctx, &s3.PutObjectInput{
		Bucket: aws.String(mgr.TargetS3.Bucket),
		Key:    aws.String(tgtKey),
		Body:   readSeeker,
	})
	if err != nil {
		return fmt.Errorf("failed to put object: %v", err)
	}
	atomic.AddInt64(&mgr.Progress.CompletedBytes, *mgr.SourceHead.ContentLength)
	return nil
}

func (mgr *MigrationMgrStruct) MigrateObject(size int64) error {
	log.Printf("Considering the migration of object %s/%s => %s/%s with size %d (%s)\n", mgr.SourceS3.Bucket, mgr.SourceS3.ObjectKey, mgr.TargetS3.Bucket, mgr.TargetS3.ObjectKey, size, HumanReadableStorageCapacity(size))
	if size > defaultPartSizeMax*10000 {
		return fmt.Errorf("content length of %d is too large to process; api limitation exceeded", size)
	}

	// skip if the object is too big
	if size > mgr.SourceS3.skipSize && mgr.SourceS3.skipSize > 0 {
		log.Printf("Skipping object s3://%s/%s, as it exceeds imposed size of limit of ( %d bytes ) %s\n", mgr.SourceS3.Bucket, mgr.SourceS3.ObjectKey, mgr.SourceS3.skipSize, HumanReadableStorageCapacity(mgr.SourceS3.skipSize))
		atomic.AddInt64(&mgr.Progress.SkippedObjects, 1)
		return fmt.Errorf("skip size exceeded")
	}

	VerbosePrintln("--- past size skip check ---")
	// skip if the target already has the object and it's newer

	//any prerequistites / checks etc
	err := withRetry(func() error {
		var err error
		mgr.SourceHead, err = mgr.SourceS3.Client.HeadObjectWithContext(mgr.SourceS3.ctx, &s3.HeadObjectInput{
			Bucket: aws.String(mgr.SourceS3.Bucket),
			Key:    aws.String(mgr.SourceS3.ObjectKey),
		})
		return err
	})

	VerbosePrintln("--- got headers of source ---")

	if err != nil {
		return fmt.Errorf("failed to get source object details: %v", err)
	}

	VerbosePrintln("--- does the target exist already? ---")

	if mgr.TargetS3 == nil {
		panic("TargetS3 is nil")
	}

	if mgr.TargetS3.ObjectExists() {
		//object exists, see if it's newer
		log.Printf("Target object s3://%s/%s already exists\n", mgr.TargetS3.Bucket, mgr.TargetS3.ObjectKey)
		mgr.TargetHead, err = mgr.TargetS3.Client.HeadObjectWithContext(mgr.TargetS3.ctx, &s3.HeadObjectInput{
			Bucket: aws.String(mgr.TargetS3.Bucket),
			Key:    aws.String(mgr.TargetS3.ObjectKey),
		})
		if err != nil {
			return fmt.Errorf("failed to get target object details: %v", err)
		}
		if !GetForce() && tgtComesAfterSrc(*mgr.TargetHead, *mgr.SourceHead) {
			log.Printf("Skipping object s3://%s/%s, target is newer than source\n", mgr.TargetS3.Bucket, mgr.TargetS3.ObjectKey)
			atomic.AddInt64(&mgr.Progress.SkippedObjects, 1)
			return nil
		}
		VerbosePrintln("--- target exists, but it's older... moving on ---")

	}

	VerbosePrintln("--- if size is less than defaultPartSizeMin, go regular, otherwise, go MPU ---")

	// determine if using MPU or not; if so, go MPU function otherwise regular copy
	if size < defaultPartSizeMin {
		VerbosePrintln("--- going regular copy ---")
		if err := mgr.CopyObjectBetweenBucketsRegular(); err != nil {
			return err
		}
	} else {
		VerbosePrintln("--- going MPU copy ---")
		if err := mgr.CopyObjectBetweenBucketsMPU(); err != nil {
			return err
		}
	}
	log.Printf("Migration of object (%s/%s) => (%s/%s) is complete", mgr.SourceS3.Bucket, mgr.SourceS3.ObjectKey, mgr.TargetS3.Bucket, mgr.TargetS3.ObjectKey)
	//handle errors, record results, return error if needed
	atomic.AddInt64(&mgr.Progress.MigratedObjects, 1)

	//bytes already counted in the copy function

	return nil
}

func (mgr *MigrationMgrStruct) IsDone() bool {
	if mgr.output == nil {
		return true
	}
	return !aws.BoolValue(mgr.output.IsTruncated)
}
