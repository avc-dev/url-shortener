// Package api содержит proto-определения gRPC-сервисов.
//
// Для перегенерации Go-кода из shortener.proto выполните из корня репозитория:
//
//	go generate ./api/...
//
// Требования: protoc, protoc-gen-go, protoc-gen-go-grpc.
// Установка:
//
//	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
//	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
//
// Путь к системным .proto-включениям задаётся переменной окружения PROTO_INCLUDE.
// По умолчанию используется $HOME/.local/include (стандартный путь при ручной установке protoc).
// Пример для Linux с системным protoc:
//
//	PROTO_INCLUDE=/usr/local/include go generate ./api/...
package api

//go:generate protoc --proto_path=. --proto_path=${PROTO_INCLUDE:-$HOME/.local/include} --go_out=../internal/proto --go_opt=paths=source_relative --go-grpc_out=../internal/proto --go-grpc_opt=paths=source_relative shortener.proto
