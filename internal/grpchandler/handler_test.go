package grpchandler_test

import (
	"context"
	"net"
	"testing"

	"github.com/avc-dev/url-shortener/internal/grpchandler"
	"github.com/avc-dev/url-shortener/internal/mocks"
	"github.com/avc-dev/url-shortener/internal/model"
	pb "github.com/avc-dev/url-shortener/internal/proto"
	"github.com/avc-dev/url-shortener/internal/service"
	"github.com/avc-dev/url-shortener/internal/usecase"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/emptypb"
)

const bufSize = 1024 * 1024

// testServer содержит всё необходимое для gRPC-тестов.
type testServer struct {
	client      pb.ShortenerServiceClient
	mockUsecase *mocks.MockURLUsecase
	authService *service.AuthService
}

// newTestServer поднимает gRPC-сервер на bufconn и возвращает клиент к нему.
// bufconn — in-memory listener из стандартной gRPC-библиотеки; не занимает реальный TCP-порт.
func newTestServer(t *testing.T) *testServer {
	t.Helper()

	mockUsecase := mocks.NewMockURLUsecase(t)
	authService := service.NewAuthService("test-secret-key")

	lis := bufconn.Listen(bufSize)
	srv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			grpchandler.LoggingInterceptor(zap.NewNop()),
			grpchandler.AuthInterceptor(authService),
		),
	)
	pb.RegisterShortenerServiceServer(srv, grpchandler.New(mockUsecase))

	t.Cleanup(func() { srv.Stop() })
	go srv.Serve(lis) //nolint:errcheck

	conn, err := grpc.NewClient(
		"passthrough://bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })

	return &testServer{
		client:      pb.NewShortenerServiceClient(conn),
		mockUsecase: mockUsecase,
		authService: authService,
	}
}

// authCtx возвращает контекст с валидным JWT в metadata-заголовке "authorization".
func (ts *testServer) authCtx(t *testing.T, userID string) context.Context {
	t.Helper()
	token, err := ts.authService.GenerateJWT(userID)
	require.NoError(t, err)
	return metadata.NewOutgoingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+token))
}

// ─── ShortenURL ──────────────────────────────────────────────────────────────

func TestShortenURL_Success(t *testing.T) {
	ts := newTestServer(t)

	ts.mockUsecase.EXPECT().
		CreateShortURLFromString("https://example.com", "user-123").
		Return("http://localhost:8080/abc12345", nil).Once()

	resp, err := ts.client.ShortenURL(ts.authCtx(t, "user-123"), &pb.URLShortenRequest{Url: "https://example.com"})
	require.NoError(t, err)
	assert.Equal(t, "http://localhost:8080/abc12345", resp.GetResult())
}

func TestShortenURL_AnonymousUser(t *testing.T) {
	// ShortenURL работает без токена: URL создаётся под сгенерированным анонимным user_id.
	ts := newTestServer(t)

	ts.mockUsecase.EXPECT().
		CreateShortURLFromString("https://example.com", mock.AnythingOfType("string")).
		Return("http://localhost:8080/abc12345", nil).Once()

	resp, err := ts.client.ShortenURL(context.Background(), &pb.URLShortenRequest{Url: "https://example.com"})
	require.NoError(t, err)
	assert.Equal(t, "http://localhost:8080/abc12345", resp.GetResult())
}

func TestShortenURL_PlainToken(t *testing.T) {
	// Токен без префикса "Bearer " тоже должен обрабатываться корректно.
	ts := newTestServer(t)
	token, err := ts.authService.GenerateJWT("user-plain")
	require.NoError(t, err)

	ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("authorization", token))

	ts.mockUsecase.EXPECT().
		CreateShortURLFromString("https://example.com", "user-plain").
		Return("http://localhost:8080/abc12345", nil).Once()

	resp, err := ts.client.ShortenURL(ctx, &pb.URLShortenRequest{Url: "https://example.com"})
	require.NoError(t, err)
	assert.Equal(t, "http://localhost:8080/abc12345", resp.GetResult())
}

func TestShortenURL_InvalidURL(t *testing.T) {
	ts := newTestServer(t)

	ts.mockUsecase.EXPECT().
		CreateShortURLFromString("not-a-url", mock.AnythingOfType("string")).
		Return("", usecase.ErrInvalidURL).Once()

	_, err := ts.client.ShortenURL(context.Background(), &pb.URLShortenRequest{Url: "not-a-url"})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestShortenURL_EmptyURL(t *testing.T) {
	ts := newTestServer(t)

	ts.mockUsecase.EXPECT().
		CreateShortURLFromString("", mock.AnythingOfType("string")).
		Return("", usecase.ErrEmptyURL).Once()

	_, err := ts.client.ShortenURL(context.Background(), &pb.URLShortenRequest{Url: ""})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestShortenURL_URLAlreadyExists(t *testing.T) {
	ts := newTestServer(t)

	ts.mockUsecase.EXPECT().
		CreateShortURLFromString("https://example.com", mock.AnythingOfType("string")).
		Return("", usecase.URLAlreadyExistsError{Code: "http://localhost:8080/existing"}).Once()

	_, err := ts.client.ShortenURL(context.Background(), &pb.URLShortenRequest{Url: "https://example.com"})
	require.Error(t, err)
	assert.Equal(t, codes.AlreadyExists, status.Code(err))
	assert.Contains(t, status.Convert(err).Message(), "http://localhost:8080/existing")
}

// ─── ExpandURL ───────────────────────────────────────────────────────────────

func TestExpandURL_Success(t *testing.T) {
	ts := newTestServer(t)

	ts.mockUsecase.EXPECT().
		GetOriginalURL("abc12345").
		Return("https://example.com", nil).Once()

	resp, err := ts.client.ExpandURL(context.Background(), &pb.URLExpandRequest{Id: "abc12345"})
	require.NoError(t, err)
	assert.Equal(t, "https://example.com", resp.GetResult())
}

func TestExpandURL_NotFound(t *testing.T) {
	ts := newTestServer(t)

	ts.mockUsecase.EXPECT().
		GetOriginalURL("unknown").
		Return("", usecase.ErrURLNotFound).Once()

	_, err := ts.client.ExpandURL(context.Background(), &pb.URLExpandRequest{Id: "unknown"})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestExpandURL_Deleted(t *testing.T) {
	ts := newTestServer(t)

	ts.mockUsecase.EXPECT().
		GetOriginalURL("deleted").
		Return("", usecase.ErrURLDeleted).Once()

	_, err := ts.client.ExpandURL(context.Background(), &pb.URLExpandRequest{Id: "deleted"})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

// ─── ListUserURLs ─────────────────────────────────────────────────────────────

func TestListUserURLs_Success(t *testing.T) {
	ts := newTestServer(t)

	ts.mockUsecase.EXPECT().
		GetURLsByUserID("user-123").
		Return([]model.UserURLResponse{
			{ShortURL: "http://localhost:8080/abc", OriginalURL: "https://example.com"},
			{ShortURL: "http://localhost:8080/def", OriginalURL: "https://google.com"},
		}, nil).Once()

	resp, err := ts.client.ListUserURLs(ts.authCtx(t, "user-123"), &emptypb.Empty{})
	require.NoError(t, err)
	require.Len(t, resp.GetUrl(), 2)
	assert.Equal(t, "http://localhost:8080/abc", resp.GetUrl()[0].GetShortUrl())
	assert.Equal(t, "https://example.com", resp.GetUrl()[0].GetOriginalUrl())
}

func TestListUserURLs_Empty(t *testing.T) {
	ts := newTestServer(t)

	ts.mockUsecase.EXPECT().
		GetURLsByUserID("user-123").
		Return([]model.UserURLResponse{}, nil).Once()

	resp, err := ts.client.ListUserURLs(ts.authCtx(t, "user-123"), &emptypb.Empty{})
	require.NoError(t, err)
	assert.Empty(t, resp.GetUrl())
}

func TestListUserURLs_NoToken_Unauthenticated(t *testing.T) {
	// Без токена ListUserURLs должен возвращать Unauthenticated:
	// клиент не передал JWT, а сервер не может угадать, чьи URL вернуть.
	ts := newTestServer(t)

	_, err := ts.client.ListUserURLs(context.Background(), &emptypb.Empty{})
	require.Error(t, err)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestListUserURLs_InvalidToken_Unauthenticated(t *testing.T) {
	ts := newTestServer(t)

	ctx := metadata.NewOutgoingContext(context.Background(),
		metadata.Pairs("authorization", "Bearer invalid.token.here"))

	_, err := ts.client.ListUserURLs(ctx, &emptypb.Empty{})
	require.Error(t, err)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}
