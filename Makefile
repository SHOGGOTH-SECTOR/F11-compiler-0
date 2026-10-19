.PHONY: build test clean
build:
	go build -o f11c ./cmd/f11c

test: build
	./f11c examples/hello.f11 -o /tmp/f11-hello
	/tmp/f11-hello; test $$? -eq 8
	./f11c examples/escape.f11 -o /tmp/f11-escape
	/tmp/f11-escape; test $$? -eq 42
	! ./f11c --emit check examples/closure_error.f11

clean:
	rm -f f11c hello escape *.ll *.o
