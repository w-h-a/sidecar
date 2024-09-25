FROM golang
WORKDIR /service
COPY service.go .
RUN go mod init service
RUN go mod tidy
RUN CGO_ENABLED=0 go build -o /service ./
CMD ["./service"]