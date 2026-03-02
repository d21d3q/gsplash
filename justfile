default: build

build:
    mkdir -p build
    CGO_ENABLED=1 go build -trimpath -ldflags "-s -w" -o build/gsplash .

install:
    mkdir -p build
    CGO_ENABLED=1 go build -trimpath -ldflags "-s -w" -o build/gsplash .
    install -d /usr/bin
    install -m 0755 build/gsplash /usr/bin/gsplash
