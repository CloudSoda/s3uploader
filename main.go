package main

import (
	"bytes"
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

const (
	maxPartSize = int64(5 * 1024 * 1024)
	maxRetries  = 10
)

var (
	awsAccessKeyID     string
	awsSecretAccessKey string
	awsBucketName      string
	apiURL             string
	filename           string
	region             string
)

func init() {
	flag.StringVar(&awsAccessKeyID, "access-key", os.Getenv("AWS_ACCESS_KEY_ID"), "S3 Access Key ID")
	flag.StringVar(&awsSecretAccessKey, "secret-key", os.Getenv("AWS_SECRET_ACCESS_KEY"), "S3 Secret Access Key")
	flag.StringVar(&awsBucketName, "bucket", os.Getenv("AWS_BUCKET_NAME"), "S3 Bucket Name")
	flag.StringVar(&apiURL, "api-url", os.Getenv("API_URL"), "S3 API URL")
	flag.StringVar(&region, "region", os.Getenv("REGION"), "S3 Region")
	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Println("USAGE:\t etag [flags] <file>")
		flag.PrintDefaults()
		os.Exit(1)
	}
	filename = flag.Arg(0)
}

func main() {
	if awsAccessKeyID == "" || awsSecretAccessKey == "" || awsBucketName == "" || apiURL == "" || region == "" {
		fmt.Println("Error: AWS Access Key ID, Secret Access Key, Bucket Name, API URL and Region are required.")
		flag.PrintDefaults()
		os.Exit(1)
	}

	fmt.Printf("File to upload: %s\n", filename)

	creds := credentials.NewStaticCredentials(awsAccessKeyID, awsSecretAccessKey, "")
	_, err := creds.Get()
	if err != nil {
		fmt.Printf("Bad credentials: %s\n", err)
		os.Exit(1)
	}

	cfg := aws.NewConfig().WithCredentials(creds).WithEndpoint(apiURL).WithRegion(region)

	sess, err := session.NewSession(cfg)
	if err != nil {
		fmt.Printf("Error creating session: %s\n", err)
		os.Exit(1)
	}

	svc := s3.New(sess)

	file, err := os.Open(filename)
	if err != nil {
		fmt.Printf("Error opening file: %s\n", err)
		os.Exit(1)
	}
	defer file.Close()

	fileInfo, _ := file.Stat()
	size := fileInfo.Size()
	buffer := make([]byte, size)
	fileType := http.DetectContentType(buffer)
	file.Read(buffer)

	path := "/multipartupload/" + file.Name()
	input := &s3.CreateMultipartUploadInput{
		Bucket:      aws.String(awsBucketName),
		Key:         aws.String(path),
		ContentType: aws.String(fileType),
	}

	resp, err := svc.CreateMultipartUpload(input)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Println("Created multipart upload request")

	var curr, partLength int64
	var remaining = size
	var completedParts []*s3.CompletedPart
	partNumber := 1
	for curr = 0; remaining != 0; curr += partLength {
		if remaining < maxPartSize {
			partLength = remaining
		} else {
			partLength = maxPartSize
		}
		completedPart, err := uploadPart(svc, resp, buffer[curr:curr+partLength], partNumber)
		if err != nil {
			fmt.Println(err.Error())
			err := abortMultipartUpload(svc, resp)
			if err != nil {
				fmt.Println(err.Error())
			}
			return
		}
		remaining -= partLength
		partNumber++
		completedParts = append(completedParts, completedPart)
	}

	completeResponse, err := completeMultipartUpload(svc, resp, completedParts)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	fmt.Printf("Successfully uploaded file: %s\n", completeResponse.String())
}

func completeMultipartUpload(svc *s3.S3, resp *s3.CreateMultipartUploadOutput, completedParts []*s3.CompletedPart) (*s3.CompleteMultipartUploadOutput, error) {
	completeInput := &s3.CompleteMultipartUploadInput{
		Bucket:   resp.Bucket,
		Key:      resp.Key,
		UploadId: resp.UploadId,
		MultipartUpload: &s3.CompletedMultipartUpload{
			Parts: completedParts,
		},
	}
	return svc.CompleteMultipartUpload(completeInput)
}

func valueOr[A any](a *A, d A) A {
	if a == nil {
		return d
	}
	return *a
}

func uploadPart(svc *s3.S3, resp *s3.CreateMultipartUploadOutput, fileBytes []byte, partNumber int) (*s3.CompletedPart, error) {
	tryNum := 1
	partInput := &s3.UploadPartInput{
		Body:          bytes.NewReader(fileBytes),
		Bucket:        resp.Bucket,
		Key:           resp.Key,
		PartNumber:    aws.Int64(int64(partNumber)),
		UploadId:      resp.UploadId,
		ContentLength: aws.Int64(int64(len(fileBytes))),
	}

	for tryNum <= maxRetries {
		uploadResult, err := svc.UploadPart(partInput)
		if err != nil {
			if tryNum == maxRetries {
				if aerr, ok := err.(awserr.Error); ok {
					return nil, aerr
				}
				return nil, err
			}
			fmt.Printf("Retrying to upload part #%v\n", partNumber)
			tryNum++
		} else {
			fmt.Printf("Uploaded part #%v, ETag: %s\n", partNumber, valueOr(uploadResult.ETag, "<nil>"))
			return &s3.CompletedPart{
				ETag:       uploadResult.ETag,
				PartNumber: aws.Int64(int64(partNumber)),
			}, nil
		}
	}
	return nil, nil
}

func abortMultipartUpload(svc *s3.S3, resp *s3.CreateMultipartUploadOutput) error {
	fmt.Println("Aborting multipart upload for UploadId#" + *resp.UploadId)
	abortInput := &s3.AbortMultipartUploadInput{
		Bucket:   resp.Bucket,
		Key:      resp.Key,
		UploadId: resp.UploadId,
	}
	_, err := svc.AbortMultipartUpload(abortInput)
	return err
}
