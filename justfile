default: build

build:
    GOARCH=arm GOOS=linux CGO_ENABLED=1 CC=arm-linux-gnueabi-gcc go build --ldflags '-linkmode external -extldflags=-static' -o build/gsplash .