# Start the Go app build
FROM golang:latest AS build

# Copy source
WORKDIR /src
COPY /src /src

# Get required modules (assumes packages have been added to ./vendor)
RUN go mod download

# Build a statically-linked Go binary for Linux
RUN CGO_ENABLED=0 GOOS=linux go build -a -o main .


# New build phase -- create binary-only image
FROM alpine:latest

# Add support for HTTPS
RUN apk update && \
    apk upgrade && \
    apk add ca-certificates

WORKDIR /

# Copy files from previous build container
COPY --from=build /src/main ./

EXPOSE 8080

# Add environment variables
# ENV ...
ENV AWS_REGION=us-east-1
ENV AWS_ACCESS_KEY_ID=AKIA34XNLPJYLB5S3VOQ
ENV AWS_SECRET_ACCESS_KEY=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
ENV LOGGLY_TOKEN=54653701-3393-48ab-b29d-3a9aee7784ab

# Check results
RUN env && pwd && find .

# Start the application
CMD ["./main"]
