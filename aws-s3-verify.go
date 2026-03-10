package alfredo

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go/aws"

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
)

type ObjectIter interface {
	Next() (*s3.Object, error)
}

type S3Iter struct {
	svc   s3iface.S3API
	input *s3.ListObjectsV2Input
	page  []*s3.Object
	idx   int
	done  bool
}

func NewS3Iter(s3c *S3ClientSession, input *s3.ListObjectsV2Input) *S3Iter {
	if err := s3c.EstablishSession(); err != nil {
		fmt.Printf("error establishing session for bucket %s: %v\n", s3c.Bucket, err)
		return &S3Iter{svc: nil, input: input}
	}

	return &S3Iter{svc: s3c.Client, input: input}
}

func (it *S3Iter) Next() (*s3.Object, error) {
	VerbosePrintf("BEGIN S3Iter.Next()")
	defer VerbosePrintln("END S3Iter.Next()")
	if it.done {
		return nil, nil
	}
	if it.page == nil || it.idx >= len(it.page) {
		if it.svc == nil {
			return nil, fmt.Errorf("S3Iter has nil svc")
		}
		if it.input == nil {
			return nil, fmt.Errorf("S3Iter has nil input")
		}

		out, err := it.svc.ListObjectsV2(it.input)
		if err != nil {
			return nil, err
		}
		it.page = out.Contents
		it.idx = 0
		if out.IsTruncated != nil && *out.IsTruncated {
			it.input.ContinuationToken = out.NextContinuationToken
		} else {
			it.done = true
		}
		if len(it.page) == 0 {
			return nil, nil
		}
	}
	obj := it.page[it.idx]
	it.idx++
	return obj, nil
}

type DiffType string

const (
	Missing      DiffType = "missing"
	Extra        DiffType = "extra"
	SizeMismatch DiffType = "size_mismatch"
)

type Diff struct {
	Type       DiffType `json:"type"`
	Key        string   `json:"key"`
	SourceSize int64    `json:"source_size,omitempty"`
	TargetSize int64    `json:"target_size,omitempty"`
}

func RunVerification(srcs3c, tgts3c *S3ClientSession, UseSourceAsPrefixOnTarget bool, out io.Writer) error {
	VerbosePrintf("BEGIN RunVerification(srcs3c!=nil %t, tgts3c!=nil %t, UseSourceAsPrefixOnTarget=%t)", srcs3c != nil, tgts3c != nil, UseSourceAsPrefixOnTarget)
	defer VerbosePrintln("END RunVerification()")
	enc := json.NewEncoder(out)

	srcIter := NewS3Iter(srcs3c, &s3.ListObjectsV2Input{
		Bucket: aws.String(srcs3c.Bucket),
	})
	if srcIter.svc == nil {
		return fmt.Errorf("error establishing session for source bucket %s", srcs3c.Bucket)
	}
	prefix := ""
	if UseSourceAsPrefixOnTarget {
		prefix = srcs3c.Bucket + "/"
	} else {
		prefix = ""
	}

	dstIter := NewS3Iter(tgts3c, &s3.ListObjectsV2Input{
		Bucket: aws.String(tgts3c.Bucket),
		Prefix: aws.String(prefix),
	})

	src, err := srcIter.Next()
	if err != nil {
		return err
	}
	dst, err := dstIter.Next()
	if err != nil {
		return err
	}

	for src != nil || dst != nil {
		var sk, dk string
		if src != nil {
			sk = *src.Key
		}
		if dst != nil {
			if UseSourceAsPrefixOnTarget {
				dk = strings.TrimPrefix(*dst.Key, srcs3c.Bucket+"/")
			} else {
				dk = *dst.Key
			}
		}

		switch {
		case src != nil && (dst == nil || sk < dk):
			enc.Encode(Diff{Type: Missing, Key: sk, SourceSize: *src.Size})
			src, err = srcIter.Next()
		case dst != nil && (src == nil || dk < sk):
			enc.Encode(Diff{Type: Extra, Key: dk, TargetSize: *dst.Size})
			dst, err = dstIter.Next()
		default:
			if *src.Size != *dst.Size {
				enc.Encode(Diff{
					Type:       SizeMismatch,
					Key:        sk,
					SourceSize: *src.Size,
					TargetSize: *dst.Size,
				})
			}
			src, err = srcIter.Next()
			if err != nil {
				return err
			}
			dst, err = dstIter.Next()
		}
		if err != nil {
			return err
		}
	}
	return nil
}
