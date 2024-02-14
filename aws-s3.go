package alfredo

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

type S3credStruct struct {
	AccessKey  string `json:"accessKey"`
	Active     bool   `json:"active"`
	CreateDate int64  `json:"createDate"`
	ExpireDate int64  `json:"expireDate"`
	SecretKey  string `json:"secretKey"`
	Profile    string `json:"profile"`
}

type S3ClientSession struct {
	Credentials S3credStruct `json:"s3creds"`
	Bucket      string
	Endpoint    string
	Region      string
	Client      *s3.S3
	Versioning  bool
	established bool
	keepBucket  bool
}

//lint:ignore ST1006 no reason
func (this S3ClientSession) SetEndpoint(sep string) S3ClientSession {
	this.Endpoint = sep

	if !strings.HasPrefix(this.Endpoint, "http") {
		this.Endpoint = "https://" + this.Endpoint
	}

	if strings.HasPrefix(this.Endpoint, "https://s3-") {
		dotidx := strings.Index(this.Endpoint, ".")
		this.Region = this.Endpoint[len("https://s3-"):dotidx]
		VerbosePrintln("region=" + this.Region)
	} else if strings.HasPrefix(this.Endpoint, "http://s3-") {
		dotidx := strings.Index(this.Endpoint, ".")
		this.Region = this.Endpoint[len("http://s3-"):dotidx]
		VerbosePrintln("region=" + this.Region)
	} else {
		VerbosePrintln("endpoint is missing http[s]://s3-; ep is: " + this.Endpoint)
	}
	return this
}

//lint:ignore ST1006 no reason
func (this S3ClientSession) SetRegion(r string) S3ClientSession {
	this.Region = r
	return this
}

//lint:ignore ST1006 no reason
func (this S3ClientSession) SetVersioning(v bool) S3ClientSession {
	this.Versioning = v
	return this
}

//lint:ignore ST1006 no reason
func (s3c S3ClientSession) KeepBucket() S3ClientSession {
	s3c.keepBucket = true
	return s3c
}

//lint:ignore ST1006 no reason
func (this S3ClientSession) EstablishSession() S3ClientSession {
	VerbosePrintln("===== establishing S3 Session =========")
	//this.sess
	ct := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	awsConfig := aws.NewConfig().
		WithEndpoint(this.Endpoint).
		WithCredentials(credentials.NewStaticCredentials(this.Credentials.AccessKey, this.Credentials.SecretKey, "")).
		WithS3ForcePathStyle(true).
		WithRegion(this.Region).
		WithHTTPClient(&http.Client{Transport: ct})

	sess := session.Must(session.NewSession(awsConfig))
	this.Client = s3.New(sess)
	this.established = true
	return this
}

//lint:ignore ST1006 no reason
func (this S3ClientSession) SetBucket(b string) S3ClientSession {
	this.Bucket = b
	return this
}

//lint:ignore ST1006 no reason
func (this S3ClientSession) RemoveBucket() error {
	if !this.established {
		this = this.EstablishSession()
	}
	output, err := this.Client.HeadBucket(&s3.HeadBucketInput{
		Bucket: aws.String(this.Bucket),
	})

	fmt.Println("output from head bucket: " + output.String())

	if err != nil {
		if strings.Contains(err.Error(), "Not Found") {
			return nil
		} else {
			panic("failed to head bucket due to err: " + err.Error())
		}
	}
	_, deleteErr := this.Client.DeleteBucket(&s3.DeleteBucketInput{
		Bucket: aws.String(this.Bucket),
	})
	return deleteErr
}

//lint:ignore ST1006 no reason
func (this S3ClientSession) HeadBucket() (bool, error) {
	if !this.established {
		this = this.EstablishSession()
	}
	_, err := this.Client.HeadBucket(&s3.HeadBucketInput{
		Bucket: aws.String(this.Bucket),
	})
	if err != nil {
		if strings.Contains(err.Error(), "Not Found") {
			return false, nil
		} else {
			return false, err
		}
	}
	return true, nil
}

//lint:ignore ST1006 no reason
func (this S3ClientSession) CreateBucket() error {
	if !this.established {
		this = this.EstablishSession()
	}
	_, err := this.Client.CreateBucket(&s3.CreateBucketInput{
		Bucket: aws.String(this.Bucket),
	})

	//aws s3api put-bucket-versioning --bucket ${bucket} --versioning-configuration Status=Enabled --endpoint-url=https://$ENDPOINT --no-verify-ssl --region region
	if this.Versioning {
		_, err = this.Client.PutBucketVersioning(&s3.PutBucketVersioningInput{
			Bucket: aws.String(this.Bucket),
			VersioningConfiguration: &s3.VersioningConfiguration{
				Status: aws.String("Enabled"),
			},
		})
	}
	return err
}

//lint:ignore ST1006 no reason
func (this S3ClientSession) Sync(localPath string) error {
	return this.SyncInner(len(localPath), localPath)
}

//lint:ignore ST1006 no reason
func (this S3ClientSession) SyncInner(trimsz int, localPath string) error {
	// Get a list of local files and subdirectories
	files, err := os.ReadDir(localPath)
	if err != nil {
		return err
	}

	for _, file := range files {
		localFilePath := filepath.Join(localPath, file.Name())
		//		s3ObjectKey := filepath.ToSlash(strings.TrimPrefix(localFilePath, localPath))
		s3ObjectKey := localFilePath[trimsz:]
		if file.IsDir() {
			// Recursively sync subdirectories
			err := this.SyncInner(trimsz, localFilePath)
			if err != nil {
				return err
			}
		} else {

			VerbosePrintln("reading from: " + localFilePath)
			VerbosePrintln("writing to : " + s3ObjectKey)

			// Upload the file to S3
			fileContent, err := os.ReadFile(localFilePath)
			if err != nil {
				return err
			}

			_, err = this.Client.PutObject(&s3.PutObjectInput{
				Bucket: aws.String(this.Bucket),
				Key:    aws.String(s3ObjectKey),
				Body:   aws.ReadSeekCloser(strings.NewReader(string(fileContent))),
			})
			if err != nil {
				return err
			}

			VerbosePrintln("Uploaded: " + s3ObjectKey)
		}
	}

	return nil
}

func (this S3ClientSession) RecursiveBucketDelete() error {

	var err error
	var b bool
	b, err = this.HeadBucket()
	if err != nil {
		return err
	}
	// bucket does not exist, return clean
	if !b {
		return nil
	}

	// List all objects in the bucket.
	listObjectsInput := &s3.ListObjectsV2Input{
		Bucket: aws.String(this.Bucket),
	}

	err = this.Client.ListObjectsV2Pages(listObjectsInput, func(page *s3.ListObjectsV2Output, lastPage bool) bool {
		for _, obj := range page.Contents {
			// Delete each object.
			deleteObjectInput := &s3.DeleteObjectInput{
				Bucket: aws.String(this.Bucket),
				Key:    obj.Key,
			}

			_, err := this.Client.DeleteObject(deleteObjectInput)
			if err != nil {
				if aerr, ok := err.(awserr.Error); ok {
					fmt.Println("AWS Error:", aerr.Code(), aerr.Message())
				} else {
					fmt.Println("Error:", err.Error())
				}
			} else {
				fmt.Printf("Deleted object: %s\n", *obj.Key)
			}
		}

		return !lastPage
	})

	if err != nil {
		fmt.Println("Error listing objects:", err)
		os.Exit(1)
	}

	if this.keepBucket {
		return nil
	}

	// Delete the bucket if it's empty (optional).
	deleteBucketInput := &s3.DeleteBucketInput{
		Bucket: aws.String(this.Bucket),
	}

	_, err = this.Client.DeleteBucket(deleteBucketInput)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if strings.Contains(aerr.Code(), "BucketNotEmpty") {
				fmt.Printf("Bucket %s is not empty; skipping deletion.\n", this.Bucket)
			} else {
				fmt.Println("Error deleting bucket:", aerr.Code(), aerr.Message())
			}
		} else {
			fmt.Println("Error:", err.Error())
		}
	} else {
		fmt.Printf("Deleted bucket: %s\n", this.Bucket)
	}
	return err
}

func (s3c S3credStruct) CredentialsStanza() string {
	return fmt.Sprintf("[%s]\naws_access_key_id = %s\naws_secret_access_key = %s\n\n",
		s3c.Profile,
		s3c.AccessKey,
		s3c.SecretKey)
}

func (s3c S3credStruct) GenerateAWSCLItoString(endpoint string, region string, useSSL bool) string {
	if strings.HasPrefix(endpoint, "http") {
		endpoint = endpoint[strings.Index(endpoint, ":")+3:]
	}
	var verify string
	var proto string
	if useSSL {
		verify = "--no-verify-ssl "
		proto = "https://"
	} else {
		verify = ""
		proto = "http://"
	}

	return fmt.Sprintf("export AWS_ACCESS_KEY_ID=%s\n", s3c.AccessKey) +
		fmt.Sprintf("export AWS_SECRET_ACCESS_KEY=%s\n", s3c.SecretKey) +
		fmt.Sprintf("export AWSOPTS=\"%s--endpoint=%s%s --region %s\"\n", verify, proto, endpoint, region) +
		fmt.Sprintln("alias cs3='aws s3 $AWSOPTS 2>/dev/null'") +
		fmt.Sprintln("alias cs3api='aws s3api $AWSOPTS 2>/dev/null'")
}

func (s3s S3ClientSession) ListBuckets() []string {
	ct := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	awsConfig := aws.NewConfig().
		WithEndpoint(s3s.Endpoint).
		WithCredentials(credentials.NewStaticCredentials(s3s.Credentials.AccessKey, s3s.Credentials.SecretKey, "")).
		WithS3ForcePathStyle(true).
		WithRegion(s3s.Region).
		WithHTTPClient(&http.Client{Transport: ct})

	sess := session.Must(session.NewSession(awsConfig))

	svc := s3.New(sess)
	input := &s3.ListBucketsInput{}

	result, err := svc.ListBuckets(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				panic(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			panic(err.Error())
		}
	}

	var returnVal []string
	if err != nil {
		panic(err.Error())
	}

	for _, bucket := range result.Buckets {
		returnVal = append(returnVal, *bucket.Name)
	}
	return returnVal
}
