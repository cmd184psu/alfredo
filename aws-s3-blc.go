package alfredo

import (
	"bytes"
	"crypto/md5"
	"crypto/tls"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
)

func (s3c *S3ClientSession) GenerateStaleMPUCleanUpRule(days int) s3.LifecycleRule {
	return s3.LifecycleRule{
		ID:     aws.String("StaleMPUCleanUp"),
		Status: aws.String("Enabled"),
		Filter: &s3.LifecycleRuleFilter{
			Prefix: aws.String(""),
		},
		AbortIncompleteMultipartUpload: &s3.AbortIncompleteMultipartUpload{
			DaysAfterInitiation: aws.Int64(int64(days)),
		},
	}
}

func (s3c *S3ClientSession) GenerateTieringRule(days int) s3.LifecycleRule {
	return s3.LifecycleRule{
		ID:     aws.String("Tiering"),
		Status: aws.String("Enabled"),
		Filter: &s3.LifecycleRuleFilter{
			Prefix: aws.String(""),
		},
		Transitions: []*s3.Transition{
			{
				Days:         aws.Int64(int64(days)),
				StorageClass: aws.String("GLACIER"),
			},
		},
	}
}

// delete all rules in one shot for the current bucket
func (s3c *S3ClientSession) DeleteEntireBucketLifeCyclePolicy() error {
	if !s3c.established {
		if err := s3c.EstablishSession(); err != nil {
			return err
		}
	}

	VerbosePrintln("\n\n\nBEGIN::s3.DeleteBucketLifeCyclePolicy()")
	// Prepare the request URL
	req, _ := s3c.Client.DeleteBucketLifecycleRequest(&s3.DeleteBucketLifecycleInput{
		Bucket: aws.String(s3c.Bucket),
	})
	VerbosePrintln("\n\n\nEND::s3.DeleteBucketLifeCyclePolicy()")

	return req.Send()
}

func calculateContentMD5(xmlBytes []byte) string {
	md5Sum := md5.Sum(xmlBytes)
	return base64.StdEncoding.EncodeToString(md5Sum[:])
}

func marshalLifecycleToXML(rules []*s3.LifecycleRule) ([]byte, error) {
	type LifecycleConfiguration struct {
		XMLName xml.Name `xml:"LifecycleConfiguration"`
		Xmlns   string   `xml:"xmlns,attr"`
		Rules   []*s3.LifecycleRule
	}

	lc := &LifecycleConfiguration{
		Xmlns: "http://s3.amazonaws.com/doc/2006-03-01/",
		Rules: rules,
	}

	buf := &bytes.Buffer{}
	enc := xml.NewEncoder(buf)
	enc.Indent("", "  ")

	start := xml.StartElement{Name: xml.Name{Local: "LifecycleConfiguration"}}
	start.Attr = []xml.Attr{{Name: xml.Name{Local: "xmlns"}, Value: lc.Xmlns}}
	if err := enc.EncodeToken(start); err != nil {
		return nil, err
	}

	for _, r := range rules {
		ruleStart := xml.StartElement{Name: xml.Name{Local: "Rule"}}
		enc.EncodeToken(ruleStart)
		// Add this in the loop over each rule
		if r.Filter != nil {
			enc.EncodeToken(xml.StartElement{Name: xml.Name{Local: "Filter"}})

			// Prefix
			if r.Filter.Prefix != nil {
				enc.EncodeElement(*r.Filter.Prefix, xml.StartElement{Name: xml.Name{Local: "Prefix"}})
			} else {
				// Always emit empty <Prefix> if nil
				enc.EncodeElement("", xml.StartElement{Name: xml.Name{Local: "Prefix"}})
			}

			enc.EncodeToken(xml.EndElement{Name: xml.Name{Local: "Filter"}})
		}

		if r.ID != nil {
			enc.EncodeElement(*r.ID, xml.StartElement{Name: xml.Name{Local: "ID"}})
		}
		if r.Status != nil {
			enc.EncodeElement(*r.Status, xml.StartElement{Name: xml.Name{Local: "Status"}})
		}

		if r.Expiration != nil && r.Expiration.Days != nil {
			enc.EncodeElement(*r.Expiration.Days, xml.StartElement{Name: xml.Name{Local: "Expiration"}})
		}

		for _, t := range r.Transitions {
			transStart := xml.StartElement{Name: xml.Name{Local: "Transition"}}
			enc.EncodeToken(transStart)
			if t.Days != nil {
				enc.EncodeElement(*t.Days, xml.StartElement{Name: xml.Name{Local: "Days"}})
			}
			if t.StorageClass != nil {
				enc.EncodeElement(*t.StorageClass, xml.StartElement{Name: xml.Name{Local: "StorageClass"}})
			}
			enc.EncodeToken(xml.EndElement{Name: xml.Name{Local: "Transition"}})
		}

		if r.AbortIncompleteMultipartUpload != nil && r.AbortIncompleteMultipartUpload.DaysAfterInitiation != nil {
			abortStart := xml.StartElement{Name: xml.Name{Local: "AbortIncompleteMultipartUpload"}}
			enc.EncodeToken(abortStart)
			enc.EncodeElement(*r.AbortIncompleteMultipartUpload.DaysAfterInitiation,
				xml.StartElement{Name: xml.Name{Local: "DaysAfterInitiation"}})
			enc.EncodeToken(xml.EndElement{Name: xml.Name{Local: "AbortIncompleteMultipartUpload"}})
		}

		enc.EncodeToken(xml.EndElement{Name: xml.Name{Local: "Rule"}})
	}

	enc.EncodeToken(xml.EndElement{Name: start.Name})
	enc.Flush()
	return buf.Bytes(), nil
}

func (s3c *S3ClientSession) PutLifecycleRules(rules []*s3.LifecycleRule) error {
	return s3c.PutLifecycleRulesCustomHeaders(rules, map[string]string{})
}

// PutLifecycleRules uses the AWS SDK to apply a slice of LifecycleRules
func (s3c *S3ClientSession) PutLifecycleRulesCustomHeaders(rules []*s3.LifecycleRule, customHeaders map[string]string) error {
	if !s3c.established {
		if err := s3c.EstablishSession(); err != nil {
			return err
		}
	}

	//if no headers, use the standard SDK way to put the lifecycle rules
	if len(customHeaders) == 0 {
		input := &s3.PutBucketLifecycleConfigurationInput{
			Bucket: aws.String(s3c.Bucket),
			LifecycleConfiguration: &s3.BucketLifecycleConfiguration{
				Rules: rules,
			},
		}
		_, err := s3c.Client.PutBucketLifecycleConfiguration(input)
		if err != nil {
			return fmt.Errorf("failed to put lifecycle configuration: %w", err)
		}
		return nil
	}

	// if we have headers, do it the "hard" way:

	// Prepare the request URL
	url := fmt.Sprintf("%s/%s", s3c.Endpoint, s3c.Bucket)
	//	blc.NS = xml.Attr{Value: "http://s3.amazonaws.com/doc/2006-03-01/"}
	// Prepare the request body (empty for bucket creation)
	requestBody, err := marshalLifecycleToXML(rules)
	if err != nil {
		return err
	}

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
	for name, value := range customHeaders {
		VerbosePrintf("name: %q, value: %q", name, value)
		req.Header.Set(name, value)
	}

	req.Header.Set("Content-MD5", contentMD5)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Authorization", "AWS "+s3c.Credentials.AccessKey+":"+GenerateSignature(stringToSign, s3c.Credentials.SecretKey))

	// Execute the HTTP request
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: !s3c.enforceCertificates},
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
	VerbosePrintln("\n\n\nEND::s3.customheaderput()")

	return nil
}

func (s3c *S3ClientSession) RemoveLifecycleRule(ruleID string) error {
	if !s3c.established {
		if err := s3c.EstablishSession(); err != nil {
			return err
		}
	}

	resp, err := s3c.Client.GetBucketLifecycleConfiguration(&s3.GetBucketLifecycleConfigurationInput{
		Bucket: aws.String(s3c.Bucket),
	})
	if err != nil {
		return fmt.Errorf("get lifecycle: %w", err)
	}

	// Filter out rule by ID
	var newRules []*s3.LifecycleRule
	for _, r := range resp.Rules {
		if *r.ID != ruleID {
			newRules = append(newRules, r)
		}
	}

	return s3c.PutLifecycleRules(newRules)
}

func (s3c *S3ClientSession) AddLifecycleRule(newRule *s3.LifecycleRule, customHeaders map[string]string) error {
	if !s3c.established {
		if err := s3c.EstablishSession(); err != nil {
			return err
		}
	}

	resp, err := s3c.Client.GetBucketLifecycleConfiguration(&s3.GetBucketLifecycleConfigurationInput{
		Bucket: aws.String(s3c.Bucket),
	})
	if err != nil {
		// No rules yet
		if aerr, ok := err.(awserr.Error); ok && aerr.Code() == "NoSuchLifecycleConfiguration" {
			resp = &s3.GetBucketLifecycleConfigurationOutput{}
		} else {
			return err
		}
	}

	// Avoid duplicate ID
	var exists bool
	for _, r := range resp.Rules {
		if *r.ID == *newRule.ID {
			exists = true
			break
		}
	}
	if !exists {
		resp.Rules = append(resp.Rules, newRule)
	}

	return s3c.PutLifecycleRulesCustomHeaders(resp.Rules, customHeaders)
}
