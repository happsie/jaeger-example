FROM golang:alpine
ENV GO111MODULE=on CGO_ENABLED=0 GOOS=linux GOARCH=amd64
WORKDIR /build
COPY . .
RUN go mod download
RUN go build -o app .
WORKDIR /dist
RUN cp /build/app .
EXPOSE 8080
CMD ["/dist/app"]
