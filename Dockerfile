FROM golang:alpine as build
WORKDIR /go/src/app
COPY . .
RUN go get -d -v ./... && go install -v ./...

FROM alpine:latest
COPY --from=build /go/bin/app /bin/ddns
RUN apk add --update --no-cache ca-certificates
CMD ["/bin/ddns"]
