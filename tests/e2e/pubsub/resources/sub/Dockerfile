FROM golang
WORKDIR /service
COPY ./sub/service.go .
RUN go mod init service
RUN go get "github.com/w-h-a/pkg/sidecar@v0.34.1-rc.30"
RUN go mod tidy
RUN CGO_ENABLED=0 go build -o /service ./
CMD ["./service"]