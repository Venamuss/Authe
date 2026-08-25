package grpc

import (
	"context"

	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	model "authe/internal/user"
	userV1 "authe/pkg/proto/user/v1"
)

type Service interface {
	Get(context.Context, int) (*model.User, error)
}

type TokenManager interface {
	ExtractClaimsWithMap(tokenString string) (jwt.MapClaims, error)
	GetTrueToken(tokenString string) string
}

type Server struct {
	service      Service
	tokenManager TokenManager
	userV1.UnimplementedUserServiceServer
}

func NewServer(service Service, tokenManager TokenManager) *Server {
	return &Server{
		service:      service,
		tokenManager: tokenManager,
	}
}

func (s *Server) GetUserByID(ctx context.Context, req *userV1.GetUserByIDRequest) (*userV1.GetUserByIDResponse, error) {
	id := req.GetId()

	user, err := s.service.Get(ctx, int(id))
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get user")
	}

	var response userV1.GetUserByIDResponse

	response.User = &userV1.User{
		Id:        int64(user.ID),
		Username:  user.Username,
		Email:     user.Email,
		CreatedAt: timestamppb.New(user.CreatedAt),
		UpdatedAt: timestamppb.New(user.UpdatedAt),
	}

	return &response, nil
}

func (s *Server) VerifyToken(ctx context.Context, req *userV1.VerifyTokenRequest) (*userV1.VerifyTokenResponse, error) {
	token := req.GetToken()

	claims, err := s.tokenManager.ExtractClaimsWithMap(s.tokenManager.GetTrueToken(token))
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid token")
	}
	username, ok := claims["username"].(string)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "invalid token")
	}

	return &userV1.VerifyTokenResponse{
		Valid:    true,
		Username: username,
	}, nil
}
