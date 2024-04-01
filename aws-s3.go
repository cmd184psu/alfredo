package alfredo

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

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
	PolicyId    string
}

func (s3c *S3ClientSession) SetEndpoint(sep string) {
	s3c.Endpoint = sep

	if !strings.HasPrefix(s3c.Endpoint, "http") {
		s3c.Endpoint = "https://" + s3c.Endpoint
	}

	if strings.HasPrefix(s3c.Endpoint, "https://s3-") {
		dotidx := strings.Index(s3c.Endpoint, ".")
		s3c.Region = s3c.Endpoint[len("https://s3-"):dotidx]
		VerbosePrintln("region=" + s3c.Region)
	} else if strings.HasPrefix(s3c.Endpoint, "http://s3-") {
		dotidx := strings.Index(s3c.Endpoint, ".")
		s3c.Region = s3c.Endpoint[len("http://s3-"):dotidx]
		VerbosePrintln("region=" + s3c.Region)
	} else {
		VerbosePrintln("endpoint is missing http[s]://s3-; ep is: " + s3c.Endpoint)
	}
}

func (s3c S3ClientSession) WithEndpoint(sep string) S3ClientSession {
	s3c.SetEndpoint(sep)
	return s3c
}

func (s3c *S3ClientSession) SetRegion(r string) {
	s3c.Region = r
}
func (s3c S3ClientSession) WithRegion(r string) S3ClientSession {
	s3c.Region = r
	return s3c
}

//lint:ignore ST1006 no reason
func (this S3ClientSession) SetVersioning(v bool) S3ClientSession {
	this.Versioning = v
	return this
}

func (s3c S3ClientSession) KeepBucket() S3ClientSession {
	s3c.keepBucket = true
	return s3c
}

func (s3c *S3ClientSession) EstablishSession() error {
	if s3c.established {
		return nil
	}
	VerbosePrintln("===== establishing S3 Session =========")
	//this.sess
	ct := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	if len(s3c.Endpoint) == 0 {
		return errors.New("missing endpoint")
	}

	if len(s3c.Credentials.AccessKey) == 0 {
		return errors.New("missing credentials")
	}

	if len(s3c.Region) == 0 {
		return errors.New("missing region")
	}

	awsConfig := aws.NewConfig().
		WithEndpoint(s3c.Endpoint).
		WithCredentials(credentials.NewStaticCredentials(s3c.Credentials.AccessKey, s3c.Credentials.SecretKey, "")).
		WithS3ForcePathStyle(true).
		WithRegion(s3c.Region).
		WithHTTPClient(&http.Client{Transport: ct})

	sess := session.Must(session.NewSession(awsConfig))
	s3c.Client = s3.New(sess)
	s3c.established = true
	return nil
}

func (s3c *S3ClientSession) SetBucket(b string) {
	s3c.Bucket = b
}
func (s3c S3ClientSession) WithBucket(b string) S3ClientSession {
	s3c.Bucket = b
	return s3c
}

//lint:ignore ST1006 no reason
func (this S3ClientSession) RemoveBucket() error {
	if err := this.EstablishSession(); err != nil {
		return err
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
	if err := this.EstablishSession(); err != nil {
		return false, err
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

func GenerateSignature(sts, secret string) string {
	hash := hmac.New(sha1.New, []byte(secret))
	hash.Write([]byte(sts))
	return base64.StdEncoding.EncodeToString(hash.Sum(nil))
}

func (s3c *S3ClientSession) generateSignature(date string) string {

	stringToSign := "PUT\n\n\n" + date + "\n/" + s3c.Bucket
	fmt.Printf("sts=%q\n", stringToSign)

	//	stringToSign := fmt.Sprintf("PUT\\n\\n\\n%s\\n%s", date, "/"+s3c.Bucket)
	//	stringToSign2 := fmt.Sprintf("PUT\\n\\n\\n%s\\n%s", date, "/"+s3c.Bucket)
	VerbosePrintln("stringtosign: " + stringToSign)
	//	VerbosePrintln("stringtosign2: " + stringToSign2)
	// Generate HMAC-SHA1 hash
	return GenerateSignature(stringToSign, s3c.Credentials.SecretKey)
	// hash := hmac.New(sha1.New, []byte(s3c.Credentials.SecretKey))
	// hash.Write([]byte(stringToSign))
	// return base64.StdEncoding.EncodeToString(hash.Sum(nil))
}

// function CalculateSignature() {
// 	#Put together your string to sign
// 	httpDate=`date -u +"%a, %_d %b %Y %H:%M:%S +0000" `
// 	#httpDate="Fri, 17 Apr 2020 17:47:04 +0000"
// 	#method="GET"
// 	#method=$1
// 	contentMD5=""
// 	contentType=""

// 	if [ -z "$xamzHeadersToSign" ]; then
// 		xamzHeadersToSign=""
// 	fi
// 	#resource=$2 #"/"${bucket}${objectKey}

// 	StringToSign="$method\n$contentMD5\n$contentType\n$httpDate\n${xamzHeadersToSign}${resource}";

// 	#calculate signature
// 	export signature=`echo -en ${StringToSign} | openssl sha1 -hmac ${secretAccessKey} -binary | base64`
// 	export URL=${proto}${endpoint}${resource}
// 	export AUTH="AWS ${accessKeyId}:${signature}"
// }

func (s3c *S3ClientSession) createBucketWithPolicy() error {
	VerbosePrintln("\n\n\nBEGIN::s3.createBucketWithPolicy()")

	// Prepare the request URL
	url := fmt.Sprintf("%s/%s", s3c.Endpoint, s3c.Bucket)

	// Prepare the request body (empty for bucket creation)
	var requestBody []byte

	// Create a new HTTP request
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(requestBody))
	if err != nil {
		return err
	}

	// Set the custom header
	now := time.Now().UTC()

	// Format the date according to RFC1123
	date := now.Format(time.RFC1123)
	// Concatenate the string to sign

	// 	StringToSign="$method\n$contentMD5\n$contentType\n$httpDate\n${xamzHeadersToSign}${resource}";

	//Wed, 27 Mar 2024 21:12:45 UTC
	date = strings.ReplaceAll(date, "UTC", "+0000")

	req.Header.Set("Date", date)
	req.Header.Set("x-gmt-policyid", s3c.PolicyId)
	req.Header.Set("Authorization", "AWS "+s3c.Credentials.AccessKey+":"+s3c.generateSignature(date))
	// Execute the HTTP request
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Check the response status code
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	VerbosePrintln("\n\n\nEND::s3.createBucketWithPolicy()")

	return nil
}

func (s3c *S3ClientSession) CreateBucket() error {
	VerbosePrintln("\n\n\nBEGIN::s3.CreateBucket()")
	if err := s3c.EstablishSession(); err != nil {
		VerbosePrintln("error establishing session")

		return err
	}

	var err error
	VerbosePrintln(fmt.Sprintf("--- about to create bucket with policy: %q ---- ", s3c.PolicyId))
	VerbosePrintln("CreateBucket()::::region is " + s3c.Region)

	if len(s3c.PolicyId) == 0 || strings.EqualFold(s3c.PolicyId, "default") {
		_, err = s3c.Client.CreateBucket(&s3.CreateBucketInput{
			Bucket: aws.String(s3c.Bucket),
		})
	} else {
		return s3c.createBucketWithPolicy()
	}
	//aws s3api put-bucket-versioning --bucket ${bucket} --versioning-configuration Status=Enabled --endpoint-url=https://$ENDPOINT --no-verify-ssl --region region
	if s3c.Versioning {
		_, err = s3c.Client.PutBucketVersioning(&s3.PutBucketVersioningInput{
			Bucket: aws.String(s3c.Bucket),
			VersioningConfiguration: &s3.VersioningConfiguration{
				Status: aws.String("Enabled"),
			},
		})
	}
	if err != nil {
		VerbosePrintln("error:::: " + err.Error())
		VerbosePrintln(fmt.Sprintf("bucket was %q", s3c.Bucket))
		VerbosePrintln(fmt.Sprintf("ep was %q", s3c.Endpoint))
	}

	VerbosePrintln("END::s3.CreateBucket()")
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
	VerbosePrintln("BEGIN: RecursiveBucketDelete()")
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

	VerbosePrintln("bucket=" + this.Bucket)

	err = this.Client.ListObjectsV2Pages(listObjectsInput, func(page *s3.ListObjectsV2Output, lastPage bool) bool {
		VerbosePrintln("inside ListObjectsV2Pages")
		VerbosePrintln(fmt.Sprintf("len(page.Content)=%d", len(page.Contents)))
		for _, obj := range page.Contents {
			// Delete each object.
			deleteObjectInput := &s3.DeleteObjectInput{
				Bucket: aws.String(this.Bucket),
				Key:    obj.Key,
			}
			def := *obj.Key
			VerbosePrintln(fmt.Sprintf("delete object: %s", def))
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
	VerbosePrintln("attempt : DeleteBucket()")

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
	VerbosePrintln("END: RecursiveBucketDelete()")

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

func S3HelperScript(profile string, region string, endpoint string) string {
	var scriptLine []string
	scriptLine = append(scriptLine, "#!/usr/bin/env bash")
	scriptLine = append(scriptLine, "export S3_ENDPOINT="+endpoint)
	scriptLine = append(scriptLine, "export AWS_OPTS=\" --endpoint-url="+endpoint+" --profile "+profile+" --region "+region+"\"")
	scriptLine = append(scriptLine, "export S3API=0")
	scriptLine = append(scriptLine, "if [ \"$1\" == \"s3api\" ];	then")
	scriptLine = append(scriptLine, "\tshift")
	scriptLine = append(scriptLine, "\texport S3API=1")
	scriptLine = append(scriptLine, "fi")
	scriptLine = append(scriptLine, "if [ $S3API -eq	1 ]; then")
	scriptLine = append(scriptLine, "\taws $AWS_OPTS s3api $@")
	scriptLine = append(scriptLine, "else")
	scriptLine = append(scriptLine, "\taws $AWS_OPTS s3 $@")
	scriptLine = append(scriptLine, "fi")
	return strings.Join(scriptLine, "\n")
}

func (s3c S3ClientSession) IsVersioningEnabled() bool {
	if err := s3c.EstablishSession(); err != nil {
		panic(err.Error())
	}

	// Get the versioning status of the S3 bucket

	ct := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	VerbosePrintln(fmt.Sprintf("ak=%q,sk=%q,b=%q", s3c.Credentials.AccessKey, s3c.Credentials.SecretKey, s3c.Bucket))
	awsConfig := aws.NewConfig().
		WithEndpoint(s3c.Endpoint).
		WithCredentials(credentials.NewStaticCredentials(s3c.Credentials.AccessKey, s3c.Credentials.SecretKey, "")).
		WithS3ForcePathStyle(true).
		WithRegion(s3c.Region).
		WithHTTPClient(&http.Client{Transport: ct})

	sess := session.Must(session.NewSession(awsConfig))

	svc := s3.New(sess)
	input := &s3.GetBucketVersioningInput{Bucket: aws.String(s3c.Bucket)}

	// s3s.Client.SigningRegion = s3s.Region
	// VerbosePrintln("endpoint = " + s3s.Endpoint)
	// s3s.Client.Endpoint = s3s.Endpoint
	result, err := svc.GetBucketVersioning(input)

	if err != nil {
		panic(err.Error())
	}

	// Check if versioning is enabled
	return strings.EqualFold(aws.StringValue(result.Status), "Enabled")
}
