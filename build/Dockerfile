FROM golang:1.18-alpine AS api-build

RUN apk add --no-cache git gcc musl-dev

WORKDIR /build

COPY . .

RUN go mod download
RUN CGO=1 go build -o freedom cmd/serve/main.go

FROM alpine

WORKDIR /usr/bin

COPY --from=api-build /build/freedom .

RUN chmod a+x /usr/bin/freedom
RUN mkdir -p /opt/freedom

EXPOSE 80
EXPOSE 443

ENV APP_ENV production

ENTRYPOINT ["/usr/bin/freedom"]
