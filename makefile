
build: src/main.go
	(cd src && go build -o ../bin/main)

install:
	cp .env.example .env
