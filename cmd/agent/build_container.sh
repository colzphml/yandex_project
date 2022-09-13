CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o agent .
docker build --tag agent . -f ./Dockerfile.scratch