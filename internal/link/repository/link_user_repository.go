package repository

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
	"github.com/evan0120-yo/linkchat-go/internal/link/model"
	"google.golang.org/api/iterator"
)

type LinkUserRepository interface {
	WithTx(tx *firestore.Transaction) LinkUserRepository
	CreateLinkUser(ctx context.Context, user *model.LinkUser) error
	UpdateLinkUser(ctx context.Context, user *model.LinkUser) error
	FindLinkUserByID(ctx context.Context, id string) (*model.LinkUser, error)

	// FindByIDs 批次查詢 (已支援分批處理，無數量上限)
	FindByIDs(ctx context.Context, ids []string) ([]*model.LinkUser, error)

	SearchByDisplayName(ctx context.Context, name string) ([]*model.LinkUser, error)
}

type firestoreLinkUserRepository struct {
	client *firestore.Client
	tx     *firestore.Transaction
}

func NewLinkUserRepository(client *firestore.Client) LinkUserRepository {
	return &firestoreLinkUserRepository{client: client}
}

func (r *firestoreLinkUserRepository) WithTx(tx *firestore.Transaction) LinkUserRepository {
	return &firestoreLinkUserRepository{
		client: r.client,
		tx:     tx,
	}
}

// CreateLinkUser (Upsert)
func (r *firestoreLinkUserRepository) CreateLinkUser(ctx context.Context, user *model.LinkUser) error {
	docRef := r.client.Collection("link_users").Doc(user.ID)

	if r.tx != nil {
		return r.tx.Set(docRef, user)
	}

	_, err := docRef.Set(ctx, user)
	if err != nil {
		return fmt.Errorf("failed to create link user: %w", err)
	}
	return nil
}

// UpdateLinkUser (Upsert)
func (r *firestoreLinkUserRepository) UpdateLinkUser(ctx context.Context, user *model.LinkUser) error {
	docRef := r.client.Collection("link_users").Doc(user.ID)

	if r.tx != nil {
		return r.tx.Set(docRef, user)
	}

	_, err := docRef.Set(ctx, user)
	if err != nil {
		return fmt.Errorf("failed to update link user: %w", err)
	}
	return nil
}

// FindLinkUserByID (單筆查詢)
func (r *firestoreLinkUserRepository) FindLinkUserByID(ctx context.Context, id string) (*model.LinkUser, error) {
	docRef := r.client.Collection("link_users").Doc(id)

	docSnap, err := docRef.Get(ctx)
	if err != nil {
		return nil, err
	}

	var user model.LinkUser
	if err := docSnap.DataTo(&user); err != nil {
		return nil, err
	}
	return &user, nil
}

// FindByIDs (批次查詢 - 列表用)
// 修正：加入分批查詢邏輯，解決 Firestore IN Query 30 筆限制
func (r *firestoreLinkUserRepository) FindByIDs(ctx context.Context, ids []string) ([]*model.LinkUser, error) {
	if len(ids) == 0 {
		return []*model.LinkUser{}, nil
	}

	// 1. 去除重複 ID (避免浪費查詢額度)
	uniqueIDs := make(map[string]struct{})
	var cleanIDs []string
	for _, id := range ids {
		if _, exists := uniqueIDs[id]; !exists {
			uniqueIDs[id] = struct{}{}
			cleanIDs = append(cleanIDs, id)
		}
	}

	// 2. 分批處理 (Firestore 限制每次 IN 查詢最多 30 筆)
	const batchSize = 30
	var users []*model.LinkUser

	for i := 0; i < len(cleanIDs); i += batchSize {
		end := i + batchSize
		if end > len(cleanIDs) {
			end = len(cleanIDs)
		}

		batchIDs := cleanIDs[i:end]

		// 執行該批次查詢
		iter := r.client.Collection("link_users").Where("id", "in", batchIDs).Documents(ctx)

		for {
			doc, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				iter.Stop() // 發生錯誤時記得關閉 Iterator
				return nil, fmt.Errorf("failed to iterate link users: %w", err)
			}

			var user model.LinkUser
			err = doc.DataTo(&user)
			if err != nil {
				continue
			}
			users = append(users, &user)
		}
		iter.Stop() // 批次結束關閉 Iterator
	}

	return users, nil
}

// SearchByDisplayName (搜尋 - Prefix Search)
func (r *firestoreLinkUserRepository) SearchByDisplayName(ctx context.Context, name string) ([]*model.LinkUser, error) {
	iter := r.client.Collection("link_users").
		Where("display_name", ">=", name).
		Where("display_name", "<=", name+"\uf8ff").
		Limit(20).
		Documents(ctx)
	defer iter.Stop()

	var users []*model.LinkUser
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("search link users failed: %w", err)
		}

		var user model.LinkUser
		if err := doc.DataTo(&user); err != nil {
			continue
		}

		if user.IsActive {
			users = append(users, &user)
		}
	}
	return users, nil
}
