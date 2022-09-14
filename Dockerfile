FROM golang:alpine as build

ARG COMPONENT

RUN apk --no-cache add tzdata

WORKDIR /app

ADD ./cmd/$COMPONENT/*.go ./cmd/$COMPONENT/
ADD ./cmd/$COMPONENT/*.yaml ./cmd/$COMPONENT/
ADD ./internal/ ./internal/

COPY go.mod go.sum ./
RUN go mod download && go mod verify

WORKDIR /app/cmd/$COMPONENT

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o app .


FROM scratch as final

ARG COMPONENT

COPY --from=build /app/cmd/$COMPONENT/app /
COPY --from=build /usr/share/zoneinfo /usr/share/zoneinfo
ADD ./cmd/$COMPONENT/*.yaml /

ENV TZ=Europe/Moscow

CMD ["/app"]
