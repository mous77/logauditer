all: clean build

build:
	# build 
	export GOOS=linux
	go build -o logauditer/server server/main.go
	go build -o logauditer/cli client/main.go

	# add static
	cp -r server/static logauditer/static
	cp server/*.html logauditer/
	cp server/*.js logauditer/
	cp README.MD logauditer/


clean:
	rm -rf logauditer/*
