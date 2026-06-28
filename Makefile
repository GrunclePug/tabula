include config.mk

.PHONY: all build clean run

TAILWIND_CMD ?= bin/tailwindcss

all: build

$(TAILWIND_CMD):
	mkdir -p bin
	curl -sLO https://github.com/tailwindlabs/tailwindcss/releases/latest/download/tailwindcss-linux-x64
	chmod +x tailwindcss-linux-x64
	mv tailwindcss-linux-x64 $(TAILWIND_CMD)

tailwind: $(TAILWIND_CMD)
	$(TAILWIND_CMD) -i web/static/css/input.css -o web/static/css/style.css --minify

build: tailwind
	go build -ldflags "-X main.version=$(VERSION)" -o bin/tabula ./cmd/tabula

run: build
	./bin/tabula

clean:
	rm -rf bin/ web/static/css/style.css
