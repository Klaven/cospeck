build:
	go build -o out/cospeck main.go

run:
	go run main.go

compile:
	echo "Compiling for every OS and Platform"
	GOOS=linux GOARCH=arm go build -o out/cospeck-linux-arm main.go
	GOOS=linux GOARCH=arm64 go build -o out/cospeck-linux-arm64 main.go
	GOOS=freebsd GOARCH=386 go build -o out/cospeck-freebsd-386 main.go

install:
    go build -o ${GOPATH}/bin/cospeck main.go