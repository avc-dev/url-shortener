package grpchandler

import (
	"context"
	"errors"

	"github.com/avc-dev/url-shortener/internal/audit"
	"github.com/avc-dev/url-shortener/internal/middleware"
	"github.com/avc-dev/url-shortener/internal/model"
	pb "github.com/avc-dev/url-shortener/internal/proto"
	"github.com/avc-dev/url-shortener/internal/usecase"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// URLUsecase определяет интерфейс бизнес-логики, используемой gRPC-хендлером.
// Совпадает с подмножеством handler.URLUsecase, чтобы оба хендлера были
// фасадами над одним usecase без дублирования логики.
type URLUsecase interface {
	CreateShortURLFromString(urlString string, userID string) (string, error)
	GetOriginalURL(code string) (string, error)
	GetURLsByUserID(userID string) ([]model.UserURLResponse, error)
}

// Handler реализует ShortenerServiceServer и делегирует вызовы в URLUsecase.
// Использует audit.Notifier из пакета audit — единственный источник истины для
// этого интерфейса, что устраняет дублирование с handler.Auditor.
type Handler struct {
	pb.UnimplementedShortenerServiceServer
	usecase  URLUsecase
	auditors []audit.Notifier
}

// New создаёт новый gRPC-хендлер. Параметры auditors опциональны.
func New(uc URLUsecase, auditors ...audit.Notifier) *Handler {
	return &Handler{usecase: uc, auditors: auditors}
}

// ShortenURL реализует rpc ShortenURL — создаёт сокращённый URL.
func (h *Handler) ShortenURL(ctx context.Context, req *pb.URLShortenRequest) (*pb.URLShortenResponse, error) {
	userID, _ := middleware.GetUserIDFromContext(ctx)

	shortURL, err := h.usecase.CreateShortURLFromString(req.GetUrl(), userID)
	if err != nil {
		return nil, mapError(err)
	}

	h.emitAudit(ctx, audit.NewEvent(audit.ActionShorten, userID, req.GetUrl()))
	return &pb.URLShortenResponse{Result: shortURL}, nil
}

// ExpandURL реализует rpc ExpandURL — возвращает оригинальный URL по короткому коду.
func (h *Handler) ExpandURL(ctx context.Context, req *pb.URLExpandRequest) (*pb.URLExpandResponse, error) {
	originalURL, err := h.usecase.GetOriginalURL(req.GetId())
	if err != nil {
		return nil, mapError(err)
	}

	userID, _ := middleware.GetUserIDFromContext(ctx)
	h.emitAudit(ctx, audit.NewFollowEvent(userID, req.GetId(), originalURL))
	return &pb.URLExpandResponse{Result: originalURL}, nil
}

// ListUserURLs реализует rpc ListUserURLs — возвращает все URL текущего пользователя.
// Требует валидного JWT-токена в metadata: анонимные запросы возвращают Unauthenticated.
func (h *Handler) ListUserURLs(ctx context.Context, _ *emptypb.Empty) (*pb.UserURLsResponse, error) {
	if !IsAuthenticated(ctx) {
		return nil, status.Error(codes.Unauthenticated, "valid authorization token required")
	}

	userID, _ := middleware.GetUserIDFromContext(ctx)

	urls, err := h.usecase.GetURLsByUserID(userID)
	if err != nil {
		return nil, mapError(err)
	}

	data := make([]*pb.URLData, 0, len(urls))
	for _, u := range urls {
		data = append(data, &pb.URLData{
			ShortUrl:    u.ShortURL,
			OriginalUrl: u.OriginalURL,
		})
	}

	return &pb.UserURLsResponse{Url: data}, nil
}

// emitAudit уведомляет всех аудиторов о событии.
func (h *Handler) emitAudit(ctx context.Context, event audit.Event) {
	for _, a := range h.auditors {
		a.Notify(ctx, event)
	}
}

// mapError преобразует ошибки usecase в gRPC status-коды.
func mapError(err error) error {
	switch {
	case errors.Is(err, usecase.ErrInvalidURL), errors.Is(err, usecase.ErrEmptyURL):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, usecase.ErrURLNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, usecase.ErrURLDeleted):
		return status.Error(codes.NotFound, "URL deleted")
	default:
		var existsErr usecase.URLAlreadyExistsError
		if errors.As(err, &existsErr) {
			return status.Error(codes.AlreadyExists, existsErr.ExistingCode())
		}
		return status.Error(codes.Internal, "internal server error")
	}
}
