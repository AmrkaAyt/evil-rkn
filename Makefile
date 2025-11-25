PROTO_SRC=proto/blockchecker.proto
PROTO_OUT=proto/gen

gen:
	mkdir -p $(PROTO_OUT)
	protoc \
	  -I proto \
	  -I third_party/googleapis \
	  --go_out $(PROTO_OUT) --go_opt=paths=source_relative \
	  --go-grpc_out $(PROTO_OUT) --go-grpc_opt=paths=source_relative \
	  --grpc-gateway_out $(PROTO_OUT) --grpc-gateway_opt=paths=source_relative \
	  --grpc-gateway_opt=generate_unbound_methods=true \
	  $(PROTO_SRC)
