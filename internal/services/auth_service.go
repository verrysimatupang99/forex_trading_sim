package services

import (
	"errors"
	"time"

	"gorm.io/gorm"

	"forex-trading-sim/internal/models"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	db *gorm.DB
}

func NewAuthService(db *gorm.DB) *AuthService {
	return &AuthService{db: db}
}

type RegisterInput struct {
	Email     string `json:"email" binding:"required,email"`
	Password  string `json:"password" binding:"required,min=8"`
	FirstName string `json:"first_name" binding:"required"`
	LastName  string `json:"last_name" binding:"required"`
}

type LoginInput struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type AuthResponse struct {
	User        *models.User `json:"user"`
	AccessToken string       `json:"access_token"`
}

func (s *AuthService) Register(input RegisterInput) (*AuthResponse, error) {
	// Check if user already exists
	var existingUser models.User
	if err := s.db.Where("email = ?", input.Email).First(&existingUser).Error; err == nil {
		return nil, errors.New("user with this email already exists")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.New("failed to hash password")
	}

	// Create user
	user := models.User{
		Email:        input.Email,
		PasswordHash: string(hashedPassword),
		FirstName:    input.FirstName,
		LastName:     input.LastName,
		IsActive:     true,
		Role:         "user",
	}

	if err := s.db.Create(&user).Error; err != nil {
		return nil, errors.New("failed to create user")
	}

	// Create demo account for new user
	account := models.Account{
		UserID:        user.ID,
		AccountNumber: generateAccountNumber(),
		Balance:       10000.00, // Starting demo balance
		Equity:        10000.00,
		Leverage:      100,
		Currency:      "USD",
		IsDemo:        true,
		Status:        "active",
	}

	if err := s.db.Create(&account).Error; err != nil {
		return nil, errors.New("failed to create account")
	}

	// Generate JWT token
	token, err := GenerateJWT(user.ID, user.Email, user.Role)
	if err != nil {
		return nil, errors.New("failed to generate token")
	}

	return &AuthResponse{
		User:        &user,
		AccessToken: token,
	}, nil
}

func (s *AuthService) Login(input LoginInput) (*AuthResponse, error) {
	var user models.User
	if err := s.db.Where("email = ?", input.Email).First(&user).Error; err != nil {
		return nil, errors.New("invalid email or password")
	}

	if !user.IsActive {
		return nil, errors.New("account is suspended")
	}

	// Check password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		return nil, errors.New("invalid email or password")
	}

	// Generate JWT token
	token, err := GenerateJWT(user.ID, user.Email, user.Role)
	if err != nil {
		return nil, errors.New("failed to generate token")
	}

	return &AuthResponse{
		User:        &user,
		AccessToken: token,
	}, nil
}

func (s *AuthService) RefreshToken(userID uint) (string, error) {
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return "", errors.New("user not found")
	}

	return GenerateJWT(user.ID, user.Email, user.Role)
}

func generateAccountNumber() string {
	return "DEMO" + time.Now().Format("20060102150405")
}
