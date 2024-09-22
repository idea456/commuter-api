FROM golang:latest as base

WORKDIR /app

COPY . .

RUN CGO_ENABLEd=0 GOOS=linux go build -o /commuter-api ./cmd/api/main.go


FROM scratch

COPY --from=base /commuter-api .

EXPOSE 4001

CMD ["/commuter-api"]