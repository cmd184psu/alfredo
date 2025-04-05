package alfredo

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"gopkg.in/ini.v1"
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
	Bucket      string       `json:"bucket"`
	Endpoint    string       `json:"endpoint"`
	Region      string       `json:"region"`
	ObjectKey   string       `json:"key"`
	//	Client      *s3.S3
	Client            s3iface.S3API
	Versioning        bool `json:"versioning"`
	established       bool
	keepBucket        bool
	PolicyId          string `json:"policyid"`
	session           *session.Session
	ctx               context.Context
	forceSSL          bool
	logging           bool
	maxConcurrency    int
	skipSize          int64
	Response          *http.Response
	ContinuationToken *string
	BatchSize         int `json:"batchSize"`
}

// deep copy with a clean new session
func (s3c *S3ClientSession) DeepCopy() S3ClientSession {
	var retValue S3ClientSession
	retValue.Credentials = s3c.Credentials
	retValue.Bucket = s3c.Bucket
	retValue.Endpoint = s3c.Endpoint
	retValue.Region = s3c.Region
	retValue.ObjectKey = s3c.ObjectKey
	retValue.Client = s3c.Client
	retValue.Versioning = s3c.Versioning
	retValue.established = false
	retValue.keepBucket = s3c.keepBucket
	retValue.PolicyId = s3c.PolicyId
	retValue.session = nil
	retValue.ctx = context.Background()
	retValue.forceSSL = s3c.forceSSL
	retValue.logging = s3c.logging
	retValue.maxConcurrency = s3c.maxConcurrency
	retValue.skipSize = s3c.skipSize
	retValue.Response = nil
	retValue.ContinuationToken = nil
	retValue.BatchSize = s3c.BatchSize

	if err := retValue.EstablishSession(); err != nil {
		panic(err.Error())
	}

	return retValue
}

const S3_default_credentials_file = "~/.aws/credentials"

func (s3c *S3ClientSession) SetSession(s *session.Session) {
	s3c.session = s
}

func (s3c *S3ClientSession) SetConcurrency(c int) {
	s3c.maxConcurrency = c
}
func (s3c *S3ClientSession) GetConcurrency() int {
	if s3c.maxConcurrency == 0 {
		s3c.maxConcurrency = defaultMaxConcurrency
	}
	return s3c.maxConcurrency
}

func (s3c *S3ClientSession) GetSession() *session.Session {
	return s3c.session
}

func (s3c *S3ClientSession) ClearEndpoint(sep string) {
	s3c.Endpoint = ""
}

func (s3c *S3ClientSession) ForceSSL() {
	s3c.forceSSL = true
}
func (s3c *S3ClientSession) DoNotForceSSL() {
	s3c.forceSSL = false
}

func (s3c *S3ClientSession) SetEndpoint(sep string) {
	if len(sep) == 0 {
		panic("attempted to set endpoint to blank - use ClearEndpoint() instead")
	}
	s3c.Endpoint = sep

	VerbosePrintf("!!! S3ClientSession::SetEndpoint(%s)", sep)
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

func (s3c *S3ClientSession) SetSkipSize(sz int64) {
	s3c.skipSize = sz
}

func (s3c S3ClientSession) WithSkipSize(sz int64) S3ClientSession {
	s3c.skipSize = sz
	return s3c
}

func (s3c S3ClientSession) GetSkipSize() int64 {
	return s3c.skipSize
}

func (s3c S3ClientSession) KeepBucket() S3ClientSession {
	s3c.keepBucket = true
	return s3c
}

func (s3c *S3ClientSession) EstablishSession() error {
	VerbosePrintln("BEGIN S3ClientSession::EstablishSession()")
	if s3c.established {
		return nil
	}
	VerbosePrintln("===== establishing S3 Session =========")
	//this.sess
	ct := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	if len(s3c.Endpoint) == 0 {
		return PanicError("missing endpoint")
	}

	if len(s3c.Credentials.AccessKey) == 0 {
		return errors.New("missing credentials")
	}

	if len(s3c.Region) == 0 {
		panic("missing region")
		//		return errors.New("missing region")
	}

	VerbosePrintf("!!! ep:%s, ak/sk: %s/%s, fps: %s, r: %s", s3c.Endpoint, s3c.Credentials.AccessKey, s3c.Credentials.SecretKey, TrueIsYes(true), s3c.Region)

	awsConfig := aws.NewConfig().
		WithEndpoint(s3c.Endpoint).
		WithCredentials(credentials.NewStaticCredentials(s3c.Credentials.AccessKey, s3c.Credentials.SecretKey, "")).
		WithS3ForcePathStyle(true).
		WithRegion(s3c.Region).
		WithHTTPClient(&http.Client{Transport: ct})

	s3c.SetSession(session.Must(session.NewSession(awsConfig)))
	s3c.Client = s3.New(s3c.GetSession())
	s3c.established = true
	VerbosePrintln("END S3ClientSession::EstablishSession()")
	return nil
}

func (s3c *S3ClientSession) SetBucket(b string) {
	s3c.Bucket = b
}
func (s3c S3ClientSession) WithBucket(b string) S3ClientSession {
	s3c.Bucket = b
	return s3c
}

func (s3c S3ClientSession) RemoveBucket() error {
	if err := s3c.EstablishSession(); err != nil {
		return err
	}
	if len(s3c.Bucket) == 0 {
		return errors.New("bucket is not set")
	}
	_, err := s3c.Client.HeadBucket(&s3.HeadBucketInput{
		Bucket: aws.String(s3c.Bucket),
	})

	//fmt.Println("output from head bucket: " + output.String())

	if err != nil {
		if strings.Contains(err.Error(), "Not Found") {
			return nil
		} else {
			panic("failed to head bucket due to err: " + err.Error())
		}
	}
	_, deleteErr := s3c.Client.DeleteBucket(&s3.DeleteBucketInput{
		Bucket: aws.String(s3c.Bucket),
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
	//fmt.Printf("sts=%q\n", stringToSign)

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
	s3c.Response = resp
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
	VerbosePrintf("bucket was %q", s3c.Bucket)
	VerbosePrintf("creds: %q/%q", s3c.Credentials.AccessKey, s3c.Credentials.SecretKey)
	VerbosePrintf("region: %q", s3c.Region)
	VerbosePrintf("ep: %q", s3c.Endpoint)

	VerbosePrintf("!!! ep:%s, ak/sk: %s/%s, fps: %s, r: %s", s3c.Endpoint, s3c.Credentials.AccessKey, s3c.Credentials.SecretKey, TrueIsYes(true), s3c.Region)

	if len(s3c.PolicyId) == 0 || strings.EqualFold(s3c.PolicyId, "default") {
		VerbosePrintln("s3c.Client.CreateBucket()")
		s3out, s3err := s3c.Client.CreateBucket(&s3.CreateBucketInput{
			Bucket: aws.String(s3c.Bucket),
		})

		VerbosePrintf("::::output: %q", s3out.String())

		err = s3err

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

func (s3c S3ClientSession) ListObjectsWithPrefix(prefix string) ([]string, error) {
	VerbosePrintf("BEGIN: ListObjectsWithPrefix(%s)", prefix)
	// List all objects in the bucket.
	VerbosePrintln("bucket=" + s3c.Bucket)

	resp, err := s3c.Client.ListObjectsV2(&s3.ListObjectsV2Input{
		Bucket: aws.String(s3c.Bucket),
		Prefix: aws.String(prefix),
	})

	var results []string
	if err != nil {
		return results, err
	}
	for _, item := range resp.Contents {
		results = append(results, *item.Key)
	}
	VerbosePrintf("BEGIN: ListObjectsWithPrefix(%s)", prefix)
	return results, nil
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

func (s3c S3ClientSession) RecursiveBucketDeleteAlt() error {
	VerbosePrintln("BEGIN: RecursiveBucketDelete()")

	// Get the list of objects in the bucket
	objects, err := s3c.Client.ListObjectsV2(&s3.ListObjectsV2Input{
		Bucket: aws.String(s3c.Bucket),
	})
	if err != nil {
		return err
	}

	// Delete each object recursively
	for _, obj := range objects.Contents {
		key := *obj.Key
		VerbosePrintf("Deleting %s\n", key)
		_, err = s3c.Client.DeleteObject(&s3.DeleteObjectInput{
			Bucket: aws.String(s3c.Bucket),
			Key:    obj.Key,
		})
		if err != nil {
			return err
		}
	}

	// Delete the bucket if it is empty
	if len(objects.Contents) == 0 {
		_, err = s3c.Client.DeleteBucket(&s3.DeleteBucketInput{
			Bucket: aws.String(s3c.Bucket),
		})
	}

	return err
}

func (s3c S3credStruct) CredentialsStanza() string {
	return fmt.Sprintf("[%s]\naws_access_key_id = %s\naws_secret_access_key = %s\n\n",
		s3c.Profile,
		s3c.AccessKey,
		s3c.SecretKey)
}

func (s3c S3credStruct) CredentialsS3FSPassword() string {
	return fmt.Sprintf("%s:%s\n", s3c.AccessKey, s3c.SecretKey)
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
				VerbosePrintf("(1) endpoint %s, credentials: %s/%s, region: %s", s3s.Endpoint, s3s.Credentials.AccessKey, s3s.Credentials.SecretKey, s3s.Region)
				panic(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			VerbosePrintf("(2) endpoint %s, credentials: %s/%s, region: %s", s3s.Endpoint, s3s.Credentials.AccessKey, s3s.Credentials.SecretKey, s3s.Region)
			panic(err.Error())
		}
	}

	var returnVal []string
	VerbosePrintf("resulting bucket list has %d elements", len(result.Buckets))
	for _, bucket := range result.Buckets {
		returnVal = append(returnVal, *bucket.Name)
	}
	return returnVal
}

func S3HelperScript(profile string, region string, endpoint string) string {
	var scriptLine []string
	scriptLine = append(scriptLine, "#!/usr/bin/env bash")
	scriptLine = append(scriptLine, "export S3_ENDPOINT="+endpoint)
	scriptLine = append(scriptLine, "export AWS_OPTS=\" --endpoint-url=${S3_ENDPOINT} --profile "+profile+" --region "+region+" --no-verify-ssl\"")
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

func S3HelperScriptBuiltInCreds(region string, endpoint string, ak string, sk string) string {
	var scriptLine []string
	scriptLine = append(scriptLine, "#!/usr/bin/env bash")
	scriptLine = append(scriptLine, "export AWS_ACCESS_KEY_ID="+ak)
	scriptLine = append(scriptLine, "export AWS_SECRET_ACCESS_KEY="+sk)

	scriptLine = append(scriptLine, fmt.Sprintf("export AWS_OPTS=\" --endpoint-url=%s --region %s --no-verify-ssl\"", endpoint, region))
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

func S3HelperScriptDeepClean(profile string, region string, endpoint string) string {
	return S3HelperScriptBuiltInCredsDeepCleanCommon(region, endpoint, "", "", profile)
}

func S3HelperScriptBuiltInCredsDeepClean(region string, endpoint string, ak string, sk string) string {
	return S3HelperScriptBuiltInCredsDeepCleanCommon(region, endpoint, ak, sk, "")
}

func S3HelperScriptBuiltInCredsDeepCleanCommon(region string, endpoint string, ak string, sk string, profile string) string {
	var scriptLine []string
	scriptLine = append(scriptLine, "#!/usr/bin/env bash")
	if !(len(ak) == 0 || len(sk) == 0) {
		scriptLine = append(scriptLine, "export AWS_ACCESS_KEY_ID="+ak)
		scriptLine = append(scriptLine, "export AWS_SECRET_ACCESS_KEY="+sk)
	}
	if len(profile) > 0 {
		scriptLine = append(scriptLine, fmt.Sprintf("export AWS_OPTS=\" --endpoint-url=%s --region %s --no-verify-ssl\" --profile %s", endpoint, region, profile))
	} else {
		scriptLine = append(scriptLine, fmt.Sprintf("export AWS_OPTS=\" --endpoint-url=%s --region %s --no-verify-ssl\"", endpoint, region))
	}
	scriptLine = append(scriptLine, "export BUCKET_NAME=$1")
	scriptLine = append(scriptLine, "MAX_KEYS=1000  # Max objects per request (AWS limit)")
	scriptLine = append(scriptLine, "if [ -z \"$BUCKET_NAME\" ]; then")
	scriptLine = append(scriptLine, "\techo \"Usage: $0 <bucket-name>\"")
	scriptLine = append(scriptLine, "\texit 1")
	scriptLine = append(scriptLine, "fi")
	scriptLine = append(scriptLine, "echo \"Deleting all versions and delete markers from bucket: $BUCKET_NAME\"")
	scriptLine = append(scriptLine, "while : ; do")
	scriptLine = append(scriptLine, "\t# List all versions and delete markers in the bucket")
	scriptLine= append(scriptLine, "\tOBJECTS_JSON=$(aws s3api list-object-versions ${AWS_OPTS} --bucket \"$BUCKET_NAME\" --max-items $MAX_KEYS --query '{Objects: (Versions || []), DeleteMarkers: (DeleteMarkers || [])}' --output json)")
	scriptLine = append(scriptLine, "\t# Extract objects to delete")
	scriptLine = append(scriptLine, "\tOBJECTS=$(echo \"$OBJECTS_JSON\" | jq -c '{Objects: ((.Objects + .DeleteMarkers) // [])}')")
	scriptLine = append(scriptLine, "\t# Break if there are no more objects")
	scriptLine = append(scriptLine, "if [[ \"$OBJECTS\" == '{\"Objects\":[]}' ]]; then")
	scriptLine = append(scriptLine, "\t\techo \"All object versions and delete markers have been deleted.\"")
	scriptLine = append(scriptLine, "\t\tbreak")
	scriptLine = append(scriptLine, "\tfi")
	scriptLine = append(scriptLine, "\t# Delete the batch of objects")
	scriptLine = append(scriptLine, "\t	OBJECT_COUNT=$(echo \"$OBJECTS\" | jq '.Objects | length')")
	scriptLine=append(scriptLine,"\tif [[ \"$OBJECT_COUNT\" -eq 0 ]]; then")
	scriptLine = append(scriptLine, "\t\techo \"No objects found, exiting loop.\"")
	scriptLine = append(scriptLine, "\t\tbreak")
	scriptLine = append(scriptLine, "\tfi")

	scriptLine = append(scriptLine, "echo \"Deleting $OBJECT_COUNT objects...\")")
	scriptLine = append(scriptLine, "aws s3api ${AWS_OPTS} delete-objects --bucket \"$BUCKET_NAME\" --cli-input-json \"$OBJECTS\"")
	scriptLine = append(scriptLine, "\t# small delay to avoid throttling")
	scriptLine = append(scriptLine, "\tsleep 1")
	scriptLine = append(scriptLine, "done")
	scriptLine = append(scriptLine, "echo \"Bucket cleanup completed.\"")
	scriptLine = append(scriptLine, "echo \"Deleting bucket: $BUCKET_NAME\"")
	scriptLine = append(scriptLine, "aws s3api ${AWS_OPTS} delete-bucket --bucket \"$BUCKET_NAME\"")
	scriptLine = append(scriptLine, "echo \"Bucket deleted.\"")
	scriptLine = append(scriptLine, "echo \"Done.\"")
	scriptLine = append(scriptLine, "exit 0")
	return strings.Join(scriptLine, "\n")
}

func S3HelperScriptBuiltInCredsCreateBucket(region string, endpoint string, ak string, sk string, bucket string) string {
	var scriptLine []string
	scriptLine = append(scriptLine, "export AWS_ACCESS_KEY_ID="+ak)
	scriptLine = append(scriptLine, "export AWS_SECRET_ACCESS_KEY="+sk)
	scriptLine = append(scriptLine, fmt.Sprintf("export AWS_OPTS=\" --endpoint-url=%s --region %s --no-verify-ssl\"", endpoint, region))
	scriptLine = append(scriptLine, fmt.Sprintf("\taws $AWS_OPTS s3 mb s3://%s", bucket))
	return strings.Join(scriptLine, "\n")
}

func (s3c S3ClientSession) IsVersioningEnabled() bool {
	if len(s3c.Bucket) == 0 {
		panic("bucket is not set")
	}
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

	VerbosePrintf("Versioning status=%q", aws.StringValue(result.Status))

	return strings.EqualFold(aws.StringValue(result.Status), "Enabled")
}

type AWSProfile struct {
	AccessKeyID     string `ini:"aws_access_key_id"`
	SecretAccessKey string `ini:"aws_secret_access_key"`
}

// f = ~/.aws/credentials
func (s3c *S3ClientSession) LoadCredentials(f string) error {
	filename := ExpandTilde(f)
	if !FileExistsEasy(filename) {
		return fmt.Errorf("file %s does not exist", filename)
	}
	VerbosePrintf("loading %s", filename)
	cfg, err := ini.Load(filename)
	if err != nil {
		return err
	}

	// Define a map to store profiles
	//profiles := make(map[string]AWSProfile)

	// Loop through each section in the configuration file
	for _, section := range cfg.Sections() {
		// Skip the default section
		if section.Name() == ini.DefaultSection {
			continue
		}
		VerbosePrintf("compareing %s vs %s", s3c.Credentials.Profile, section.Name())

		if strings.EqualFold(s3c.Credentials.Profile, section.Name()) {
			VerbosePrintf("\tfound!")
			// Create a new AWSProfile instance
			p := AWSProfile{}
			// Map the section to the AWSProfile struct
			if err := section.MapTo(&p); err != nil {
				VerbosePrintln("error: " + err.Error())
				return err
			}
			VerbosePrintf("ak=%s sk=%s", p.AccessKeyID, p.SecretAccessKey)
			s3c.Credentials.AccessKey = p.AccessKeyID
			s3c.Credentials.SecretKey = p.SecretAccessKey
			return nil
		} else {
			VerbosePrintf("\trejected!")
		}
	}
	return fmt.Errorf("profile %s was not found in configuration file %s", s3c.Credentials.Profile, f)
}

func (s3c *S3ClientSession) LoadUserCredentialsForProfile() error {
	return s3c.LoadCredentials(S3_default_credentials_file)
}

func (s3c *S3ClientSession) PresignedURL(expiredHours int) (string, error) {
	VerbosePrintf("BEGIN::PresignedURL(%d)", expiredHours)

	VerbosePrintf("\tbucket=%q", s3c.Bucket)
	VerbosePrintf("\tkey=%q", s3c.ObjectKey)
	if s3c.Client == nil {
		panic(errors.New("s3c.Client is nil"))
	}
	req, _ := s3c.Client.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String(s3c.Bucket),
		Key:    aws.String(s3c.ObjectKey),
	})
	result, err := req.Presign(time.Duration(expiredHours*3600) * time.Second)
	VerbosePrintf("END::PresignedURL(%d)", expiredHours)
	return result, err
}

func (s3c *S3ClientSession) GetObject() ([]byte, error) {
	// Get the object from S3
	result, err := s3c.Client.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(s3c.Bucket),
		Key:    aws.String(s3c.ObjectKey),
	})
	if err != nil {
		return make([]byte, 0), err
	}
	defer result.Body.Close()
	return io.ReadAll(result.Body)
}

func (s3c *S3ClientSession) HeadObject() (bool, error) {
	_, err := s3c.Client.HeadObject(&s3.HeadObjectInput{
		Bucket: aws.String(s3c.Bucket),
		Key:    aws.String(s3c.ObjectKey),
	})

	if err != nil {

		if strings.Contains(err.Error(), "status code: 404") {

			//if aerr, ok := err.(awserr.Error); ok && aerr.Code() == s3.ErrCodeNoSuchKey {
			// Object does not exist
			VerbosePrintln("returning false,nil because error was caught")
			return false, nil
		}
		// Some other error occurred
		VerbosePrintf("returning false,ERR because error was not caught: %q", err.Error())
		return false, err
	}

	// Object exists
	return true, nil
}

func (s3c *S3ClientSession) GetS3Ptr() s3iface.S3API {
	return s3c.Client
}

func (s3c *S3ClientSession) ObjectExists() bool {
	b, err := s3c.HeadObject()
	if err != nil {
		panic(err.Error())
	}
	return b
}

func (s3c *S3ClientSession) GetObjectHash() (string, error) {
	ba, err := s3c.GetObject()
	return MD5SumBA(ba), err
}

// expected: s3://bucket/object
func (s3c *S3ClientSession) ParseFromURL(url string) error {
	if !strings.HasPrefix(url, "s3://") {
		return errors.New("malformed URL")
	}
	parts := strings.Split(url[5:], "/")
	s3c.Bucket = parts[0]
	s3c.ObjectKey = strings.Join(parts[1:], "/")
	return nil
}

func (s3c *S3ClientSession) GetURL() string {
	return fmt.Sprintf("s3://%s/%s", s3c.Bucket, s3c.ObjectKey)
}

func (s3c *S3ClientSession) Load(filename string) error {
	if FileExistsEasy(filename) {
		if err := ReadStructFromJSONFile(filename, &s3c); err != nil {
			return err
		}
	} else {
		jsonContent := "[]"
		json.Unmarshal([]byte(jsonContent), &s3c)
	}
	return nil
}

func (s3c S3ClientSession) Save(filename string) error {
	if err := WriteStructToJSONFile(filename, s3c); err != nil {
		return err
	}
	return nil
}

// ProgressReader wraps an io.Reader and provides progress updates
type ProgressReader struct {
	io.Reader
	Total    int64
	Uploaded int64
	Key      string
}

func (r *ProgressReader) Read(p []byte) (int, error) {
	n, err := r.Reader.Read(p)
	r.Uploaded += int64(n)
	progress := float64(r.Uploaded) / float64(r.Total) * 100
	//	log.Printf("Uploading %s: %.2f%%", r.Key, progress)
	fmt.Printf("\rUploading %s: %.2f%%", r.Key, progress)
	return n, err
}
func (s3c *S3ClientSession) EnableLogging(l bool) {
	s3c.logging = l
	if s3c.logging {
		log.Println("Logging is now enabled (S3ClientSession)")
	}
}
func (s3c S3ClientSession) S3SyncDirectoryToBucket(dirPath string, progress *ProgressTracker) error {
	uploader := s3manager.NewUploader(s3c.GetSession())
	skip := false
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		skip = false
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("failed to open file %s: %v", path, err)
		}
		defer file.Close()

		key, err := filepath.Rel(dirPath, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %v", err)
		}

		// Create a progress reader
		progressReader := &ProgressReader{
			Reader: file,
			Total:  info.Size(),
			Key:    key,
		}
		s3c.ctx = context.Background()
		headOutput, herr := s3c.Client.HeadObjectWithContext(s3c.ctx, &s3.HeadObjectInput{
			Bucket: aws.String(s3c.Bucket),
			Key:    aws.String(key),
		})
		if herr == nil {
			VerbosePrintf("headOutput: Etag: %s", *headOutput.ETag)
			if !GetForce() {
				log.Printf("Skipping object s3://%s/%s, already exists on target", s3c.Bucket, key)
				skip = true

			}
		}

		if skip && !GetForce() {
			if s3c.logging {
				log.Printf("\nSkipped upload for file %s to s3://%s/%s, object already exists\n", path, s3c.Bucket, key)
			}
			fmt.Printf("\nSkipped upload for file %s to s3://%s/%s, object already exists\n", path, s3c.Bucket, key)
			VerbosePrintf("migrated/skipped before: %d/%d object:%s", progress.MigratedObjects, progress.SkippedObjects, key)
			atomic.AddInt64(&progress.SkippedObjects, 1)
			VerbosePrintf("migrated/skipped after: %d/%d object:%s", progress.MigratedObjects, progress.SkippedObjects, key)

		} else {
			_, err = uploader.Upload(&s3manager.UploadInput{
				Bucket: aws.String(s3c.Bucket),
				Key:    aws.String(key),
				Body:   progressReader,
			})
			if err != nil {
				return fmt.Errorf("failed to upload file %s: %v", path, err)
			}
			if s3c.logging {
				log.Printf("\nUploaded %s to s3://%s/%s\n", path, s3c.Bucket, key)
			}
			fmt.Printf("\nUploaded %s to s3://%s/%s\n", path, s3c.Bucket, key)
			VerbosePrintf("migrated/skipped before: %d/%d object:%s", progress.MigratedObjects, progress.SkippedObjects, key)
			atomic.AddInt64(&progress.MigratedObjects, 1)
			VerbosePrintf("migrated/skipped after: %d/%d object:%s", progress.MigratedObjects, progress.SkippedObjects, key)
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("error walking the directory: %v", err)
	}
	if s3c.logging {
		log.Printf("\nSync has concluded for bucket %s\n", s3c.Bucket)
	}

	return nil
}

const (
	defaultPartSizeMin int64 = 5 * 1024 * 1024           // 5MB per part
	defaultPartSizeMax int64 = defaultPartSizeMin * 1024 // 5GB per part
	//defaultPartSizeMax int64 = defaultPartSizeMin + 1024*1024 // 5KB+10 bytes? per part
	//maxConcurrency           = 10                      // Maximum number of concurrent part uploads
	maxRetries = 3               // Maximum number of retries for failed operations
	retryDelay = 1 * time.Second // Delay between retries
	maxParts   = 10000           // per AMZ specification
	//maxParts              = 10 // per AMZ specification
	defaultMaxConcurrency = 10
)

type ProgressTracker struct {
	TotalObjects    int64
	MigratedObjects int64
	SkippedObjects  int64
	TotalBytes      int64
	CompletedBytes  int64
	FailedObjects   map[string]error
	mu              sync.Mutex
}

func (p *ProgressTracker) Lock() {
	p.mu.Lock()
}
func (p *ProgressTracker) Unlock() {
	p.mu.Unlock()
}

type CopyResult struct {
	SourceKey   string
	TargetKey   string
	Success     bool
	Error       error
	BytesCopied int64
	Duration    time.Duration
}

// type EndpointInfo struct {
// 	Endpoint string
// 	Region   string
// 	Bucket   string
// }

func withRetry(operation func() error) error {
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		if err := operation(); err != nil {
			lastErr = err
			if aerr, ok := err.(awserr.Error); ok {
				if aerr.Code() == "NoSuchKey" || aerr.Code() == "NoSuchBucket" {
					return err
				}
			}
			time.Sleep(retryDelay * time.Duration(i+1))
			continue
		}
		return nil
	}
	return fmt.Errorf("operation failed after %d retries: %v", maxRetries, lastErr)
}

// CopyAllObjects copies all objects between S3-compatible systems
func (sourceS3 S3ClientSession) CopyAllObjectsDoNotUse(
	targetS3 *S3ClientSession,
	progress *ProgressTracker,
) error {
	sourceS3.ctx = context.Background()
	targetS3.ctx = context.Background()

	progress.FailedObjects = make(map[string]error)

	// Create worker pool for concurrent object copying
	workerPool := make(chan struct{}, sourceS3.GetConcurrency())
	var wg sync.WaitGroup
	resultsChan := make(chan CopyResult, sourceS3.GetConcurrency())

	// Start result collector
	go func() {
		for result := range resultsChan {
			if !result.Success {
				progress.mu.Lock()
				progress.FailedObjects[result.SourceKey] = result.Error
				progress.mu.Unlock()
			}
		}
	}()

	// List all objects in source bucket
	err := sourceS3.Client.ListObjectsV2PagesWithContext(sourceS3.ctx,
		&s3.ListObjectsV2Input{
			Bucket: aws.String(sourceS3.Bucket),
		},
		func(page *s3.ListObjectsV2Output, lastPage bool) bool {
			for _, obj := range page.Contents {
				atomic.AddInt64(&progress.TotalObjects, 1)
				atomic.AddInt64(&progress.TotalBytes, *obj.Size)

				key := *obj.Key
				wg.Add(1)
				go func(objectKey string) {
					defer wg.Done()
					workerPool <- struct{}{} // Acquire worker
					defer func() {
						<-workerPool // Release worker
					}()

					startTime := time.Now()
					err := sourceS3.CopyObjectBetweenBuckets(targetS3,
						objectKey, objectKey,
						progress)

					result := CopyResult{
						SourceKey:   objectKey,
						TargetKey:   objectKey,
						Success:     err == nil,
						Error:       err,
						Duration:    time.Since(startTime),
						BytesCopied: *obj.Size,
					}
					if result.Success {
						log.Printf("Uploaded object to s3://%s/%s", targetS3.Bucket, result.SourceKey)
					} else if strings.Contains(result.Error.Error(), "skip limit exceeded") {
						log.Printf("Failed to upload object to s3://%s/%s: due to skip size limit exceeded", targetS3.Bucket, result.SourceKey)
					} else {
						log.Printf("Failed to upload object to s3://%s/%s: %v", targetS3.Bucket, result.SourceKey, result.Error)
					}
					resultsChan <- result
				}(key)
			}
			return !lastPage
		})

	if err != nil {
		return fmt.Errorf("failed to list objects: %v", err)
	}

	wg.Wait()
	close(resultsChan)

	if len(progress.FailedObjects) > 0 {
		log.Printf("%s\n", PrettyPrint(progress.FailedObjects))

		return fmt.Errorf("some objects failed to copy. Check FailedObjects map for details")
	}

	return nil
}

func (sourceS3 *S3ClientSession) CopyAllObjectsBatch(
	targetS3 *S3ClientSession,
	progress *ProgressTracker, batchSize int) error {

	migrationMgr := NewMigrationManager(sourceS3, targetS3, progress, batchSize)

	var wg sync.WaitGroup
	resultsChan := make(chan CopyResult, 10000)
	i := 0
	done := make(chan bool)
	go func() {
		for result := range resultsChan {
			if !result.Success {
				progress.mu.Lock()
				progress.FailedObjects[result.SourceKey] = result.Error
				progress.mu.Unlock()
			}
		}
		done <- true
	}()

	for {
		//VerbosePrintf("loop with continuationToken: %v and %d", sourceS3.ContinuationToken, i)
		i++

		if err := migrationMgr.MigrationLoop(&wg, &resultsChan); err != nil {
			return err
		}

		if migrationMgr.IsDone() {
			break
		}
	}

	// Wait for all copy operations to complete
	wg.Wait()

	// Close results channel and wait for collector to finish
	close(resultsChan)
	<-done

	if len(progress.FailedObjects) > 0 {
		log.Printf("Failed objects: %s", PrettyPrint(progress.FailedObjects))
		return fmt.Errorf("some objects failed to copy. Check FailedObjects map for details")
	}

	return nil
}

func CalculatePartSize(objectSize int64) int64 {
	if objectSize <= defaultPartSizeMin {
		return 0
	}
	if objectSize > 10000*defaultPartSizeMax {
		return 0
	}
	partSize := defaultPartSizeMin

	for objectSize/partSize > maxParts {
		partSize *= 2
	}

	if partSize > defaultPartSizeMax {
		partSize = defaultPartSizeMax
	}
	return partSize
}

func CalculateTotalParts(objectSize, partSize int64) int64 {
	if objectSize < 5*1024*1024 {
		return 0
	}

	if objectSize < 5*1024*1024*1024 {
		return int64(math.Ceil(float64(objectSize) / float64(defaultPartSizeMin)))
	}
	return int64(math.Ceil(float64(objectSize) / float64(partSize)))
}

func tgtComesAfterSrc(tgt, src s3.HeadObjectOutput) bool {
	tgtTime := *tgt.LastModified
	srcTime := *src.LastModified
	return tgtTime.After(srcTime)
}

// CopyObjectBetweenBuckets copies a single object between S3-compatible systems
func (sourceS3 S3ClientSession) CopyObjectBetweenBuckets(
	targetS3 *S3ClientSession,
	sourceKey string,
	targetKey string,
	progress *ProgressTracker,
) error {
	// Get source object details

	var headOutputSrc *s3.HeadObjectOutput
	var headOutputTgt *s3.HeadObjectOutput
	err := withRetry(func() error {
		var err error
		headOutputSrc, err = sourceS3.Client.HeadObjectWithContext(sourceS3.ctx, &s3.HeadObjectInput{
			Bucket: aws.String(sourceS3.Bucket),
			Key:    aws.String(sourceKey),
		})
		return err
	})
	if err != nil {
		return fmt.Errorf("failed to get source object details: %v", err)
	}

	if *headOutputSrc.ContentLength > sourceS3.skipSize && sourceS3.skipSize > 0 {
		log.Printf("Skipping object s3://%s/%s, as it exceeds imposed size of limit of ( %d bytes ) %s\n", sourceS3.Bucket, sourceKey, sourceS3.skipSize, HumanReadableStorageCapacity(sourceS3.skipSize))
		atomic.AddInt64(&progress.SkippedObjects, 1)
		return fmt.Errorf("skip size exceeded")
	}

	if *headOutputSrc.ContentLength > defaultPartSizeMax*10000 {
		return fmt.Errorf("content length of %d is too large to process", *headOutputSrc.ContentLength)
	}

	headOutputTgt, err = targetS3.Client.HeadObjectWithContext(targetS3.ctx, &s3.HeadObjectInput{
		Bucket: aws.String(targetS3.Bucket),
		Key:    aws.String(targetKey),
	})
	VerbosePrintf("headOutputSrc: Etag: %s", *headOutputSrc.ETag)
	if err == nil {
		VerbosePrintf("headOutputTgt: Etag: %s", *headOutputTgt.ETag)
		if !GetForce() && tgtComesAfterSrc(*headOutputTgt, *headOutputSrc) {
			log.Printf("Skipping object s3://%s/%s, target is newer than source\n", targetS3.Bucket, targetKey)
			VerbosePrintf("migrated/skipped before: %d/%d object:%s", progress.MigratedObjects, progress.SkippedObjects, targetKey)
			atomic.AddInt64(&progress.SkippedObjects, 1)
			VerbosePrintf("migrated/skipped after: %d/%d object:%s", progress.MigratedObjects, progress.SkippedObjects, targetKey)
			atomic.AddInt64(&progress.CompletedBytes, *headOutputSrc.ContentLength)
			return nil
		}
	}

	// For small files, use GET and PUT instead of COPY
	if *headOutputSrc.ContentLength < defaultPartSizeMin {
		// Get the object
		getOutput, err := sourceS3.Client.GetObjectWithContext(sourceS3.ctx, &s3.GetObjectInput{
			Bucket: aws.String(sourceS3.Bucket),
			Key:    aws.String(sourceKey),
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
		_, err = targetS3.Client.PutObjectWithContext(sourceS3.ctx, &s3.PutObjectInput{
			Bucket: aws.String(targetS3.Bucket),
			Key:    aws.String(targetKey),
			Body:   readSeeker,
		})
		if err != nil {
			return fmt.Errorf("failed to put object: %v", err)
		}
		VerbosePrintf("migrated/skipped before: %d/%d object:%s", progress.MigratedObjects, progress.SkippedObjects, targetKey)
		atomic.AddInt64(&progress.MigratedObjects, 1)
		VerbosePrintf("migrated/skipped after: %d/%d object:%s", progress.MigratedObjects, progress.SkippedObjects, targetKey)
		atomic.AddInt64(&progress.CompletedBytes, *headOutputSrc.ContentLength)
		return nil
	}

	// For large files, use multipart upload with streaming
	log.Printf("Creating MPU for s3://%s/%s", targetS3.Bucket, targetKey)
	createOutput, err := targetS3.Client.CreateMultipartUploadWithContext(targetS3.ctx, &s3.CreateMultipartUploadInput{
		Bucket: aws.String(targetS3.Bucket),
		Key:    aws.String(targetKey),
	})
	if err != nil {
		return fmt.Errorf("failed to create multipart upload: %v", err)
	}

	partSize := CalculatePartSize(*headOutputSrc.ContentLength)
	log.Printf("Using partsize: %s", HumanReadableStorageCapacity(partSize))
	totalParts := CalculateTotalParts(*headOutputSrc.ContentLength, partSize)
	//*headOutput.ContentLength + partSize - 1) / partSize
	log.Printf("Using total parts: %d", totalParts)
	log.Printf("Maximum parts: %d", maxParts)
	if totalParts > maxParts {
		panic("requested too many parts for this object")
	}
	log.Printf("Part size range: %s-%s", HumanReadableStorageCapacity(defaultPartSizeMin), HumanReadableStorageCapacity(defaultPartSizeMax))
	parts := make([]*s3.CompletedPart, totalParts)
	partsChan := make(chan int64, totalParts)
	errorsChan := make(chan error, totalParts)
	var uploadWg sync.WaitGroup

	// Fill parts channel
	for i := int64(1); i <= int64(totalParts); i++ {
		partsChan <- i
	}
	close(partsChan)

	// Process parts concurrently
	for i := 0; i < sourceS3.GetConcurrency(); i++ {
		uploadWg.Add(1)
		go func() {
			defer uploadWg.Done()

			for partNumber := range partsChan {
				startByte := (partNumber - 1) * partSize
				endByte := startByte + partSize - 1
				if endByte >= *headOutputSrc.ContentLength {
					endByte = *headOutputSrc.ContentLength - 1
				}

				// Get the part from source
				getPartOutput, err := sourceS3.Client.GetObjectWithContext(sourceS3.ctx, &s3.GetObjectInput{
					Bucket: aws.String(sourceS3.Bucket),
					Key:    aws.String(sourceKey),
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
				log.Printf("Uploading part of MPU for s3://%s/%s part #: %d of %d", targetS3.Bucket, targetKey, partNumber, totalParts)

				uploadOutput, err := targetS3.Client.UploadPartWithContext(targetS3.ctx, &s3.UploadPartInput{
					Bucket:     aws.String(targetS3.Bucket),
					Key:        aws.String(targetKey),
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

				parts[partNumber-1] = &s3.CompletedPart{
					ETag:       uploadOutput.ETag,
					PartNumber: aws.Int64(partNumber),
				}

				atomic.AddInt64(&progress.CompletedBytes, endByte-startByte+1)
			}
		}()
	}

	uploadWg.Wait()
	close(errorsChan)

	hasErrors := false

	// Check for errors
	for err := range errorsChan {
		// Abort multipart upload
		log.Printf("Aborting MPU for s3://%s/%s due to error: %s", targetS3.Bucket, targetKey, err.Error())

		_, abortErr := targetS3.Client.AbortMultipartUploadWithContext(targetS3.ctx, &s3.AbortMultipartUploadInput{
			Bucket:   aws.String(targetS3.Bucket),
			Key:      aws.String(targetKey),
			UploadId: createOutput.UploadId,
		})
		if abortErr != nil {
			return fmt.Errorf("failed to abort multipart upload: %v (original error: %v)", abortErr, err)
		}
		hasErrors = true

	}

	if hasErrors {
		return fmt.Errorf("errors occurred; some objects MPU were aborted as a result")
	}
	log.Printf("Completing MPU for s3://%s/%s", targetS3.Bucket, targetKey)

	// Complete multipart upload
	_, err = targetS3.Client.CompleteMultipartUploadWithContext(targetS3.ctx, &s3.CompleteMultipartUploadInput{
		Bucket:   aws.String(targetS3.Bucket),
		Key:      aws.String(targetKey),
		UploadId: createOutput.UploadId,
		MultipartUpload: &s3.CompletedMultipartUpload{
			Parts: parts,
		},
	})
	if err != nil {
		log.Printf("MPU for s3://%s/%s failed to complete", targetS3.Bucket, targetKey)
		return fmt.Errorf("failed to complete multipart upload: %v", err)
	}
	log.Printf("MPU for s3://%s/%s successfully completed", targetS3.Bucket, targetKey)
	VerbosePrintf("migrated/skipped before: %d/%d object:%s", progress.MigratedObjects, progress.SkippedObjects, targetKey)
	atomic.AddInt64(&progress.MigratedObjects, 1)
	VerbosePrintf("migrated/skipped after: %d/%d object:%s", progress.MigratedObjects, progress.SkippedObjects, targetKey)
	return nil
}

type ownerBucketACLStruct struct {
	BucketOwnerID string `json:"ID"`
}
type granteeBucketACLStruct struct {
	GranteeUserID   string `json:"ID"`
	GranteeUserType string `json:"Type"`
}
type grantBucketACLStruct struct {
	Grantee    granteeBucketACLStruct `json:"Grantee"`
	Permission string                 `json:"Permission"`
}

type BucketACLStruct struct {
	Owner  ownerBucketACLStruct   `json:"Owner"`
	Grants []grantBucketACLStruct `json:"Grants"`
}

func CreateGrant(id string, perm string) grantBucketACLStruct {
	var g grantBucketACLStruct
	g.Grantee.GranteeUserID = id
	g.Permission = perm
	g.Grantee.GranteeUserType = "CanonicalUser"
	return g
}
func GenerateROBucketPolicy(existingAcl BucketACLStruct) BucketACLStruct {
	var newAcl BucketACLStruct
	newAcl.Owner.BucketOwnerID = existingAcl.Owner.BucketOwnerID

	newAcl.Grants = append(newAcl.Grants, CreateGrant(newAcl.Owner.BucketOwnerID, "READ"))
	newAcl.Grants = append(newAcl.Grants, CreateGrant(newAcl.Owner.BucketOwnerID, "WRITE_ACP"))
	newAcl.Grants = append(newAcl.Grants, CreateGrant(newAcl.Owner.BucketOwnerID, "READ_ACP"))
	return newAcl
}

func (sourceS3 S3ClientSession) GetBucketACL() (string, error) {
	VerbosePrintln("BEGIN: GetBucketACL(...)")
	if !sourceS3.established {
		return "", fmt.Errorf("S3 session was not established")
	}

	acl, err := sourceS3.Client.GetBucketAcl(&s3.GetBucketAclInput{
		Bucket: aws.String(sourceS3.Bucket),
	})
	if err != nil {
		return "", err
	}

	VerbosePrintln("END: GetBucketACL(...)")
	return PrettyPrint(acl), nil
}

func (sourceS3 S3ClientSession) SetBucketACL(aclJson string) error {
	VerbosePrintln("BEGIN: SetBucketACL()")
	if !sourceS3.established {
		return fmt.Errorf("S3 session was not established")
	}

	// Unmarshal the JSON string into an AccessControlPolicy struct
	var aclPolicy s3.AccessControlPolicy
	err := json.Unmarshal([]byte(aclJson), &aclPolicy)
	if err != nil {
		return fmt.Errorf("failed to unmarshal ACL JSON: %v", err)
	}

	// Create the PutBucketAclInput
	aclInput := &s3.PutBucketAclInput{
		Bucket:              aws.String(sourceS3.Bucket),
		AccessControlPolicy: &aclPolicy,
	}

	// Set the bucket ACL
	_, err = sourceS3.Client.PutBucketAcl(aclInput)
	if err != nil {
		return fmt.Errorf("failed to set bucket ACL: %v", err)
	}

	fmt.Printf("Successfully set ACL for bucket %s\n", sourceS3.Bucket)
	return nil
}

// fmt.Printf("MD5 hash of the byte range %d-%d: %s\n", R, R+chunk-1, hashString)
// return nil

func (s3c *S3ClientSession) GetSizeOfObject() (int64, error) {
	headObjectOutput, err := s3c.Client.HeadObject(&s3.HeadObjectInput{
		Bucket: &s3c.Bucket,
		Key:    &s3c.ObjectKey,
	})
	if err != nil {
		return 0, err
	}
	return *headObjectOutput.ContentLength, nil
}

func (s3c *S3ClientSession) GetHashOfObjectRange(fromChunk int64, chunkSize int64) (string, error) {
	size, err := s3c.GetSizeOfObject()

	if err != nil {
		return "", err
	}

	if fromChunk+chunkSize > size {
		return "", fmt.Errorf("chunk size exceeds object size")
	}

	rangeHeader := fmt.Sprintf("bytes=%d-%d", fromChunk, fromChunk+chunkSize-1)
	getObjectOutput, err := s3c.Client.GetObject(&s3.GetObjectInput{
		Bucket: &s3c.Bucket,
		Key:    &s3c.ObjectKey,
		Range:  &rangeHeader,
	})
	if err != nil {
		return "", err
	}
	defer getObjectOutput.Body.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, getObjectOutput.Body); err != nil {
		return "", err
	}
	hashInBytes := hash.Sum(nil)[:16]
	hashString := hex.EncodeToString(hashInBytes)
	return hashString, nil
}

type LifecycleConfiguration struct {
	XMLName xml.Name `xml:"LifecycleConfiguration"`
	Rules   []Rule   `xml:"Rule"`
}

// Rule represents a single rule in the lifecycle configuration
type Rule struct {
	ID                    string                        `xml:"ID,omitempty"`
	Status                string                        `xml:"Status"`
	Filter                Filter                        `xml:"Filter"`
	Transitions           []Transition                  `xml:"Transition,omitempty"`
	Expiration            *Expiration                   `xml:"Expiration,omitempty"`
	NoncurrentTransitions []NoncurrentVersionTransition `xml:"NoncurrentVersionTransition,omitempty"`
	NoncurrentExpiration  *NoncurrentVersionExpiration  `xml:"NoncurrentVersionExpiration,omitempty"`
}

// Filter represents the filter for a rule
type Filter struct {
	Prefix string `xml:"Prefix"`
}

// Transition represents a transition in a rule
type Transition struct {
	Days         int    `xml:"Days,omitempty"`
	StorageClass string `xml:"StorageClass"`
	Date         string `xml:"Date,omitempty"`
}

// Expiration represents an expiration in a rule
type Expiration struct {
	Days int    `xml:"Days,omitempty"`
	Date string `xml:"Date,omitempty"`
}

// NoncurrentVersionTransition represents a transition for noncurrent versions
type NoncurrentVersionTransition struct {
	NoncurrentDays int    `xml:"NoncurrentDays"`
	StorageClass   string `xml:"StorageClass"`
}

// NoncurrentVersionExpiration represents an expiration for noncurrent versions
type NoncurrentVersionExpiration struct {
	NoncurrentDays int `xml:"NoncurrentDays"`
}

func (s3c *S3ClientSession) DeleteBucketLifeCyclePolicy(rulename string) error {
	VerbosePrintf("\n\n\nBEGIN::s3.DeleteBucketLifeCyclePolicy(%s)", rulename)
	// Prepare the request URL
	req, _ := s3c.Client.DeleteBucketLifecycleRequest(&s3.DeleteBucketLifecycleInput{
		Bucket: aws.String(s3c.Bucket),
		//		ID:     aws.String(rulename),
	})
	VerbosePrintf("\n\n\nEND::s3.DeleteBucketLifeCyclePolicy(%s)", rulename)

	return req.Send()
}

func (s3c *S3ClientSession) GenerateLifeCycleRules(n int) (LifecycleConfiguration, error) {
	var blc LifecycleConfiguration
	blc.Rules = make([]Rule, n)
	for i := 0; i < n; i++ {
		blc.Rules[i].ID = fmt.Sprintf("rule-%d", i)
		blc.Rules[i].Filter.Prefix = ""
		blc.Rules[i].Status = "Enabled"
		blc.Rules[i].Transitions = make([]Transition, 1)
		blc.Rules[i].Transitions[0].StorageClass = "GLACIER"
		blc.Rules[i].Expiration = nil

	}
	return blc, nil
}

func calculateContentMD5(xmlBytes []byte) string {
	md5Sum := md5.Sum(xmlBytes)
	return base64.StdEncoding.EncodeToString(md5Sum[:])
}

func (blc *LifecycleConfiguration) ToByteArray() []byte {
	data, err := xml.MarshalIndent(blc, "", "  ")

	if err != nil {
		panic(err)
	}
	return []byte(xml.Header + string(data))
}

func (blc *LifecycleConfiguration) String() string {
	return string(blc.ToByteArray())
}

func (s3c *S3ClientSession) PutBucketLifeCyclePolicy(blc LifecycleConfiguration, extraHeaders map[string]string) error {
	VerbosePrintln("\n\n\nBEGIN::s3.PutBucketLifeCyclePolicy()")
	VerbosePrintf("s3c.Endpoint: %s", s3c.Endpoint)
	VerbosePrintf("s3c.Bucket: %s", s3c.Bucket)
	VerbosePrintf("s3c.Credentials.AccessKey: %s", s3c.Credentials.AccessKey)
	VerbosePrintf("s3c.Credentials.SecretKey: %s", s3c.Credentials.SecretKey)
	VerbosePrintf("s3c.Region: %s", s3c.Region)

	// Prepare the request URL
	url := fmt.Sprintf("%s/%s", s3c.Endpoint, s3c.Bucket)

	// Prepare the request body (empty for bucket creation)
	requestBody := blc.ToByteArray()

	VerbosePrintln("=======begin requestBody========")
	VerbosePrintln(string(requestBody))
	VerbosePrintln("=======end requestBody========")

	//stuff jsonData into requestBody, my marshalling and unmarshalling it
	//hash := alfredo.MD5SumString(string(requestBody))

	// Usage
	contentMD5 := calculateContentMD5(requestBody)

	fmt.Printf("contentMD5: %q\n", contentMD5)

	// Create a new HTTP request
	req, err := http.NewRequest(http.MethodPut, url+"?lifecycle", bytes.NewReader(requestBody))
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
	//	req.Header.Set("x-gmt-policyid", s3c.PolicyId)

	contentType := "text/xml"

	stringToSign := fmt.Sprintf("PUT\n%s\n%s\n%s\n/%s?lifecycle", contentMD5, contentType, date, s3c.Bucket)
	//fmt.Println("hash: ", contentMD5)
	for name, value := range extraHeaders {
		VerbosePrintf("name: %q, value: %q", name, value)
		req.Header.Set(name, value)
	}

	req.Header.Set("Content-MD5", contentMD5)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Authorization", "AWS "+s3c.Credentials.AccessKey+":"+GenerateSignature(stringToSign, s3c.Credentials.SecretKey))

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
	s3c.Response = resp
	VerbosePrintln("\n\n\nEND::s3.createbuckethw()")

	return nil
}
