GO = go
# GOFLAGS
SERVER = "server"
USER = shell whoami
BUILDDIR = "/home/toastsandwich/gocodes/epoll/http1.0_server/build"
SRC = "/home/toastsandwich/gocodes/epoll/http1.0_server/cmd/main"

.PHONY: build clean

build:
	@mkdir -p $(BUILDDIR)
	@$(GO) build -o $(BUILDDIR)/$(SERVER) $(SRC)
	@echo "Done"

clean:
	@rm -r $(BUILDDIR)
