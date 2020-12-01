# builder image
FROM golang:1.14-alpine3.11 as builder
RUN mkdir /build
ADD . /build/
RUN apk --no-cache add build-base git bzr mercurial gcc make
WORKDIR /build
RUN make


# generate clean, final image for end users
FROM alpine:3.11.3
COPY --from=builder /build/out/cospeck .
COPY --from=lachlanevenson/k8s-kubectl:v1.19.1 /usr/local/bin/kubectl /usr/local/bin/kubectl

# executable
ENTRYPOINT [ "./cospeck" ]