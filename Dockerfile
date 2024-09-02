FROM golang AS build
WORKDIR /go/src/sidecar
COPY . .
RUN CGO_ENABLED=0 go build -o /go/bin/sidecar ./

FROM alpine
RUN apk --no-cache add ca-certificates
COPY --from=build /go/bin/sidecar /bin/sidecar
ENTRYPOINT [ "/bin/sidecar" ]