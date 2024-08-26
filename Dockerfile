FROM golang AS build
WORKDIR /go/src/action
COPY . .
RUN CGO_ENABLED=0 go build -o /go/bin/action ./

FROM alpine
RUN apk --no-cache add ca-certificates
COPY --from=build /go/bin/action /bin/action
ENTRYPOINT [ "/bin/action" ]