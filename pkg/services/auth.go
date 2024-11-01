package services

type AuthService struct{}

func NewAuthService() *AuthService {
	return &AuthService{}
}

func (svc *AuthService) CreateSession() {}

func (svc *AuthService) GetSession() {}
