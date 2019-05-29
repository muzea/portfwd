#!/bin/bash
BUILD_DIR=$(dirname "$0")/build
mkdir -p $BUILD_DIR
cd $BUILD_DIR

VERSION=`date -u +%Y%m%d`
LDFLAGS="-X main.VERSION=$VERSION -s -w"
GCFLAGS=""

go get github.com/muzea/portfwd

# AMD64 
OSES=(linux darwin windows freebsd)
for os in ${OSES[@]}; do
	suffix=""
	if [ "$os" == "windows" ]
	then
		suffix=".exe"
	fi
	env CGO_ENABLED=0 GOOS=$os GOARCH=amd64 go build -ldflags "$LDFLAGS" -gcflags "$GCFLAGS" -o portfwd_${os}_amd64${suffix} github.com/muzea/portfwd
	tar -zcf portfwd-${os}-amd64-$VERSION.tar.gz portfwd_${os}_amd64${suffix}
done

# 386
OSES=(linux windows)
for os in ${OSES[@]}; do
	suffix=""
	if [ "$os" == "windows" ]
	then
		suffix=".exe"
	fi
	env CGO_ENABLED=0 GOOS=$os GOARCH=386 go build -ldflags "$LDFLAGS" -gcflags "$GCFLAGS" -o portfwd_${os}_386${suffix} github.com/muzea/portfwd
	tar -zcf portfwd-${os}-386-$VERSION.tar.gz portfwd_${os}_386${suffix}
done

# ARM
ARMS=(5 6 7)
for v in ${ARMS[@]}; do
	env CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=$v go build -ldflags "$LDFLAGS" -gcflags "$GCFLAGS" -o portfwd_linux_arm$v  github.com/muzea/portfwd
tar -zcf portfwd-linux-arm$v-$VERSION.tar.gz portfwd_linux_arm$v
done

# ARM64
env CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags "$LDFLAGS" -gcflags "$GCFLAGS" -o portfwd_linux_arm64  github.com/muzea/portfwd
tar -zcf portfwd-linux-arm64-$VERSION.tar.gz portfwd_linux_arm64

#MIPS32LE
env CGO_ENABLED=0 GOOS=linux GOARCH=mipsle GOMIPS=softfloat go build -ldflags "$LDFLAGS" -gcflags "$GCFLAGS" -o portfwd_linux_mipsle github.com/muzea/portfwd
env CGO_ENABLED=0 GOOS=linux GOARCH=mips GOMIPS=softfloat go build -ldflags "$LDFLAGS" -gcflags "$GCFLAGS" -o portfwd_linux_mips github.com/muzea/portfwd

tar -zcf portfwd-linux-mipsle-$VERSION.tar.gz portfwd_linux_mipsle
tar -zcf portfwd-linux-mips-$VERSION.tar.gz portfwd_linux_mips
