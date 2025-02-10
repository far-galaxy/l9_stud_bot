FROM golang:bookworm

WORKDIR /app

COPY . .

ENV TZ="Europe/Samara"

RUN apt update
RUN apt install -y wkhtmltopdf
ENV WK_PATH="wkhtmltoimage"

RUN go build main.go

CMD ./main