# Step 1: Build the Go binary
FROM golang:1.23.3 AS builder

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy the Go Modules manifests
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum are not changed
RUN go mod tidy

# Copy the source code into the container
COPY . .

# Build the Go app
RUN go build -o server .

# Step 2: Create the final image
FROM golang:1.23.3

# Set the Current Working Directory inside the container
WORKDIR /root/

# Copy the Go binary from the builder image
COPY --from=builder /app/server .

# Expose port 8080 to be accessible outside the container
EXPOSE 8080

# Command to run the executable
CMD ["./server"]
