.PHONY: proto data run

# This target uses a loop to iterate over the .proto files located in the 
# internal/services/*/proto/ directory. For each file ($$f), it executes 
# the protoc command to generate Go code using the gRPC plugin. 
# The --go_out=plugins=grpc:. flag specifies the output directory and the 
# plugin to use. After compilation, it echoes a message indicating that 
# the file has been successfully compiled.
proto:
	for f in internal/services/*/proto/*.proto; do \
		protoc --go_out=plugins=grpc:. $$f; \
		echo compiled: $$f; \
	done

# This target uses the go-bindata tool to generate a Go source file (bindata.go) 
# in the data package. It takes all the .json files in the data directory and 
# embeds their contents into Go code. The resulting Go file can be used to 
# access the embedded data. This target specifies the output file (-o data/bindata.go) 
# and the package name (-pkg data).
data:
	go-bindata -o data/bindata.go -pkg data data/*.json

# This target is responsible for building and running the Docker containers defined 
# in the docker-compose.yml file. It first executes docker-compose build to build the 
# container images based on the instructions in the Dockerfile(s). Then, it runs 
# docker-compose up --remove-orphans to start the containers and remove any orphaned 
# containers (containers that are no longer referenced).
run:
	docker-compose build
	docker-compose up --remove-orphans
