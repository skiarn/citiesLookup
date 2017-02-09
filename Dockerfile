# Usage 
#	Build: `docker build --tag citieslookup .`
#	Run: `docker run -p 3000:3000 -it citieslookup`
FROM golang:latest

# Copy the local package files to the container’s workspace.
ADD . /go/src/github.com/skiarn/citiesLookup

# Install api binary globally within container
RUN go install github.com/skiarn/citiesLookup

# Set binary as entrypoint
ENTRYPOINT /go/bin/citiesLookup -p=3000

# Expose default port (3000)
EXPOSE 3000
