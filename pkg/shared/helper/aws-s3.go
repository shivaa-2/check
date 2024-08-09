package helper

import (
	"bytes"
	"mime"
	"net/url"
	"os"
	"path/filepath"
	"time"

	// Additional imports needed for examples below
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

var api_key = GetenvStr("S3_API_KEY", "")
var secret = GetenvStr("S3_SECRET", "")
var endpoint = GetenvStr("S3_ENDPOINT", "")
var region = GetenvStr("S3_REGION", "")

var s3Config = &aws.Config{
	Credentials: credentials.NewStaticCredentials(api_key, secret, ""),
	Endpoint:    aws.String(endpoint),
	Region:      aws.String(region),
}
var newSession = session.New(s3Config)
var s3Client = s3.New(newSession)

func InitS3Client() {
	var api_key = os.Getenv("S3_API_KEY")
	var secret = os.Getenv("S3_SECRET")
	var endpoint = os.Getenv("S3_ENDPOINT")
	var region = os.Getenv("S3_REGION")

	var s3Config = &aws.Config{
		Credentials:      credentials.NewStaticCredentials(api_key, secret, ""),
		Endpoint:         aws.String(endpoint),
		Region:           aws.String(region),
		S3ForcePathStyle: aws.Bool(false),
	}
	var newSession = session.New(s3Config)
	s3Client = s3.New(newSession)

}

func CreateBucket(name string) bool {
	InitS3Client()
	fmt.Println(region)
	params := &s3.CreateBucketInput{
		Bucket: aws.String(name),
	}
	_, err := s3Client.CreateBucket(params)
	if err != nil {
		fmt.Println(err.Error())
		return false
	}
	return true
}

func ListBuckets() interface{} {
	InitS3Client()
	spaces, err := s3Client.ListBuckets(nil)
	if err != nil {
		fmt.Println(err.Error())
		return nil
	}
	return spaces.Buckets
	// for _, b := range spaces.Buckets {
	//     fmt.Println(aws.StringValue(b.Name))
	// }
}

func UploadFile(bucketName string, fileName string, refId string, remarks string, file []byte) (string, error) {
	InitS3Client()
	// Determine the content type based on the file extension
	ext := filepath.Ext(fileName)
	contentType := mime.TypeByExtension(ext)
	if contentType == "" {
		contentType = "application/octet-stream" // default to binary stream if unknown
	}

	// Determine the content disposition
	var contentDisposition string
	if contentType == "application/pdf" {
		contentDisposition = "attachment" // download as attachment
	} else {
		contentDisposition = "inline" // show in browser
	}
	fmt.Println(contentType, contentDisposition)

	object := s3.PutObjectInput{
		Bucket:             aws.String(bucketName),
		Key:                aws.String(fileName),
		Body:               bytes.NewReader(file),
		ContentType:        aws.String(contentType),
		ACL:                aws.String("public-read"),
		ContentDisposition: aws.String(contentDisposition),
		// Metadata: map[string]*string{
		// 	"x-amz-meta-my-key": aws.String(refId), //required
		// },
	}
	_, err := s3Client.PutObject(&object)
	if err != nil {
		return "", err
	}

	fileLink := "https://blr1.digitaloceanspaces.com/sakthipharma" + fileName
	return fileLink, nil
}

func ListBucketFiles(bucketName string) map[string]interface{} {
	input := &s3.ListObjectsInput{
		Bucket: aws.String(bucketName),
	}
	objects, err := s3Client.ListObjects(input)
	if err != nil {
		return nil
		//fmt.Println(err.Error())
	}

	result := make(map[string]interface{})
	for _, obj := range objects.Contents {
		result[*obj.Key] = obj
		//fmt.Println(aws.StringValue(obj.Key))
	}
	return result
}

func GetDownloadUrl(bucketName string, fileName string) string {
	req, _ := s3Client.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(fileName),
	})
	urlStr, err := req.Presign(5 * time.Minute)
	if err != nil {
		return ""
		//fmt.Println(err.Error())
	}
	return urlStr
}

func GetUploadUrl(bucketName string, fileName string, metaData map[string]*string) string {
	req, _ := s3Client.PutObjectRequest(&s3.PutObjectInput{
		Bucket:   aws.String(bucketName),
		Key:      aws.String(fileName),
		Metadata: metaData,
	})
	urlStr, err := req.Presign(5 * time.Minute)
	if err != nil {
		fmt.Println(err.Error())
		return ""
	}
	return urlStr
}

func DeleteFile(bucketName string, fileName string) bool {
	input := &s3.DeleteObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(fileName),
	}
	result, err := s3Client.DeleteObject(input)
	if err != nil {
		fmt.Println(err.Error())
		return false
	}
	return *result.DeleteMarker
}

func CopyFile(sourceBucketName string, targetBucketName string, filename string) bool {
	input := &s3.CopyObjectInput{
		Bucket:     aws.String(targetBucketName),
		CopySource: aws.String(url.PathEscape(sourceBucketName + "/" + filename)),
		Key:        aws.String(filename),
	}
	_, err := s3Client.CopyObject(input)
	if err != nil {
		fmt.Println(err.Error())
		return false
	}
	return true
}

func MoveFile(sourceBucketName string, targetBucketName string, filename string) bool {
	if CopyFile(sourceBucketName, targetBucketName, filename) {
		return DeleteFile(sourceBucketName, filename)
	}
	return false
}

func S3PdfFileUpload(s3Client *s3.S3, filePath string, fileName string) (string, error) {
	year := time.Now().Format("2006")
	month := time.Now().Format("01")

	pdfFolderPath := os.Getenv("PDF_FOLDER_PATH")
	FolderKey := pdfFolderPath + "/" + year + "/" + month

	// Create the folder in S3
	object := &s3.PutObjectInput{
		Bucket: aws.String("uploads"), // Replace with your S3 bucket name
		Key:    aws.String(FolderKey + "/"),
		ACL:    aws.String("public-read"),
	}

	_, err := s3Client.PutObject(object)
	if err != nil {
		fmt.Printf("Failed to create folder or upload file: %v\n", err)
		return "", err
	}
	// Open the PDF file
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Printf("Failed to open file: %v", err)
		return "", err
	}
	defer file.Close()
	fileKey := FolderKey + "/" + filepath.Base(fileName)
	// Upload the file to the created folder in S3
	object = &s3.PutObjectInput{
		Bucket:      aws.String("tpctrz"), // Replace with your S3 bucket name
		Key:         aws.String(fileKey),
		Body:        file, // Pass the opened file as the Body parameter
		ACL:         aws.String("public-read"),
		ContentType: aws.String("application/pdf"), // Set the content type to PDF
	}
	_, err = s3Client.PutObject(object)
	if err != nil {
		fmt.Printf("Failed to upload file: %v", err)
		return "", err
	}
	return fileKey, nil
}
