GOCMD=go
GOBUILD=$(GOCMD) build
BINARY_NAME=vre
MAIN_FILE=cmd/vre/main.go

vre.go:
	$(GOBUILD) -o $(BINARY_NAME) $(MAIN_FILE)

clean:
	rm $(BINARY_NAME)
