# 生成 gRPC Go 代码（需安装 protoc、protoc-gen-go、protoc-gen-go-grpc）
.PHONY: proto
proto:
	protoc --go_out=. --go_opt=module=kairos \
		--go-grpc_out=. --go-grpc_opt=module=kairos \
		-I api/proto api/proto/account.proto api/proto/social.proto
