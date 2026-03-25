.PHONY: bridge bridge-run bridge-test deploy

# Build the Go bridge binary
bridge:
	cd apps/bridge && go build -o ../../bridge .

# Run the bridge locally
bridge-run:
	cd apps/bridge && go run .

# Run bridge tests
bridge-test:
	cd apps/bridge && go test ./...

# Build for Whatbox (Linux amd64)
bridge-linux:
	cd apps/bridge && GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o ../../bridge-linux .

# Deploy bridge binary to Whatbox
deploy-bridge:
	scp -i ~/.ssh/id_ed25519_whatbox bridge-linux yolan@orange.whatbox.ca:~/bin/gaende-bridge
	ssh -i ~/.ssh/id_ed25519_whatbox yolan@orange.whatbox.ca 'chmod +x ~/bin/gaende-bridge'
