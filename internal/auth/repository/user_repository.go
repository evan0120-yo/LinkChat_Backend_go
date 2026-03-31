package repository

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
	"github.com/evan0120-yo/linkchat-go/internal/auth/model"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// interface
type UserRepository interface {
	WithTx(tx *firestore.Transaction) UserRepository
	CreateUser(ctx context.Context, user *model.User) error
	UpdateUser(ctx context.Context, user *model.User) error
	FindByEmail(ctx context.Context, email string) (*model.User, error)
	FindByID(ctx context.Context, id string) (*model.User, error)
}

// struct
type firestoreUserRepository struct {
	client *firestore.Client
	tx     *firestore.Transaction
}

// factory
func NewUserRepository(client *firestore.Client) UserRepository {
	return &firestoreUserRepository{
		client: client,
		tx:     nil,
	}
}

func (r *firestoreUserRepository) WithTx(tx *firestore.Transaction) UserRepository {
	// 複製一份 Repository，但注入 Tx
	return &firestoreUserRepository{
		client: r.client,
		tx:     tx,
	}
}

// CreateUser implements UserRepository.
func (r *firestoreUserRepository) CreateUser(ctx context.Context, user *model.User) error {
	// 1. 路徑
	docRef := r.client.Collection("users").Doc(user.ID)
	// 2. trans
	if r.tx != nil {
		return r.tx.Set(docRef, user)
	}
	// 3. normal model
	_, err := docRef.Set(ctx, user)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

// UpdateUser implements UserRepository.
func (r *firestoreUserRepository) UpdateUser(ctx context.Context, user *model.User) error {
	// 1. 路徑
	docRef := r.client.Collection("users").Doc(user.ID)
	// 2. trans
	if r.tx != nil {
		return r.tx.Set(docRef, user)
	}
	// 3. normal model
	_, err := docRef.Set(ctx, user)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}
	return nil
}

// FindByEmail implements UserRepository.
func (r *firestoreUserRepository) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	iter := r.client.Collection("users").Where("email", "==", email).Limit(1).Documents(ctx)

	doc, err := iter.Next()
	if err == iterator.Done {
		// cant find data
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find user by email: %v", err)
	}

	var user model.User
	err = doc.DataTo(&user)
	if err != nil {
		return nil, fmt.Errorf("failed to convert document to user: %v", err)
	}
	return &user, nil
}

// FindByID implements UserRepository.
func (r *firestoreUserRepository) FindByID(ctx context.Context, id string) (*model.User, error) {
	docSnap, err := r.client.Collection("users").Doc(id).Get(ctx)
	if err != nil {
		// 如果錯誤碼是 NotFound，我們回傳 (nil, nil) 表示「沒發生系統錯誤，只是沒資料」
		if status.Code(err) == codes.NotFound {
			return nil, nil
		}
		// 其他錯誤才是真的系統錯誤 (連線失敗、權限不足等)
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	var user model.User
	if err := docSnap.DataTo(&user); err != nil {
		return nil, fmt.Errorf("failed to map data: %w", err)
	}

	return &user, nil
}
