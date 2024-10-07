##
# Traefik Keymate
#
# @file
# @version 0.1

testvm:
	nixos-rebuild build-vm --flake .#test

static:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o traffikey ./cmd/server

# end
