# S3 Multipart Upload Script

The purpose of this Go program is to confirm if an object storage is returning an ETag for multipart uploads. It supports configuration through both environment variables and command-line flags.

## Requirements

- Go 1.18 or higher (for generics support)
- AWS SDK for Go v1

## Setup

1. Clone this repository to your local machine.

2. Add the required dependencies and tidy up the `go.mod` file:

   ```bash
   go mod tidy
   ```

   This command will automatically add the AWS SDK for Go and any other necessary dependencies to your project.

## Usage

### Building the Script

```bash
go build -o s3uploader main.go
```

### Running the Script

You can run the script using environment variables, command-line flags, or a combination of both.

#### Using Environment Variables

```bash
export AWS_ACCESS_KEY_ID="your-access-key"
export AWS_SECRET_ACCESS_KEY="your-secret-key"
export AWS_BUCKET_NAME="your-bucket-name"
export API_URL="https://your-api-url"
export REGION="us-west-1"
./s3uploader file-to-upload.txt
```

#### Using Command-line Flags

```bash
./s3uploader -access-key="your-access-key" -secret-key="your-secret-key" -bucket="your-bucket-name" -api-url="https://your-api-url" -region="us-west-1" file-to-upload.txt
```

#### Mixing Environment Variables and Flags

You can use a combination of environment variables and flags. Flags take precedence over environment variables.

```bash
export AWS_ACCESS_KEY_ID="your-access-key"
export AWS_SECRET_ACCESS_KEY="your-secret-key"
./s3uploader -bucket="your-bucket-name" -api-url="https://your-api-url" -region="us-west-1" file-to-upload.txt
```

### Command-line Flags

- `-access-key`: S3 Access Key ID
- `-secret-key`: S3 Secret Access Key
- `-bucket`: S3 Bucket Name
- `-api-url`: S3 API URL (useful for S3-compatible storage services)
- `-region`: S3 Bucket region

## Error Handling

The script includes error handling for:

- Missing required configuration
- Invalid credentials
- File opening errors
- Session creation errors
- Upload errors (with retry mechanism)

If an error occurs during the multipart upload process, the script will attempt to abort the multipart upload to avoid leaving incomplete uploads on S3.

## Limitations

- The maximum part size is set to 5 MB. Adjust the `maxPartSize` constant if you need larger part sizes.
- The script uses a fixed number of retries (10) for failed part uploads. Modify the `maxRetries` constant if you need a different number of retries.



## License

This script is provided under the MIT License. See the LICENSE file for details.
