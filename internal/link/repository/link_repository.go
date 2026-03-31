package repository

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
	"github.com/evan0120-yo/linkchat-go/internal/link/model"
	"google.golang.org/api/iterator"
)

type LinkRepository interface {
	WithTx(tx *firestore.Transaction) LinkRepository

	// 基本 CRUD
	CreateLink(ctx context.Context, link *model.Link) error
	UpdateLink(ctx context.Context, link *model.Link) error
	DeleteLink(ctx context.Context, linkID string) error
	FindLinkByID(ctx context.Context, id string) (*model.Link, error)

	// 列表查詢: 找出某人的所有好友關係 (包含 Pending/Active/Blocked)
	FindLinksByUserID(ctx context.Context, userID string) ([]*model.Link, error)

	// 核心檢查: 找出兩個人之間是否已有關係 (防重複檢查用)
	FindLinkByParticipants(ctx context.Context, userA, userB string) (*model.Link, error)
}

type firestoreLinkRepository struct {
	client *firestore.Client
	tx     *firestore.Transaction
}

func NewLinkRepository(client *firestore.Client) LinkRepository {
	return &firestoreLinkRepository{client: client}
}

func (r *firestoreLinkRepository) WithTx(tx *firestore.Transaction) LinkRepository {
	return &firestoreLinkRepository{
		client: r.client,
		tx:     tx,
	}
}

// CreateLink
func (r *firestoreLinkRepository) CreateLink(ctx context.Context, link *model.Link) error {
	docRef := r.client.Collection("links").Doc(link.ID)

	if r.tx != nil {
		return r.tx.Set(docRef, link)
	}

	_, err := docRef.Set(ctx, link)
	if err != nil {
		return fmt.Errorf("failed to create link: %w", err)
	}
	return nil
}

// UpdateLink
func (r *firestoreLinkRepository) UpdateLink(ctx context.Context, link *model.Link) error {
	docRef := r.client.Collection("links").Doc(link.ID)

	if r.tx != nil {
		return r.tx.Set(docRef, link)
	}

	_, err := docRef.Set(ctx, link)
	if err != nil {
		return fmt.Errorf("failed to update link: %w", err)
	}
	return nil
}

// DeleteLink
func (r *firestoreLinkRepository) DeleteLink(ctx context.Context, linkID string) error {
	docRef := r.client.Collection("links").Doc(linkID)

	if r.tx != nil {
		return r.tx.Delete(docRef)
	}

	_, err := docRef.Delete(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete link: %w", err)
	}
	return nil
}

// FindLinkByID
func (r *firestoreLinkRepository) FindLinkByID(ctx context.Context, id string) (*model.Link, error) {
	docRef := r.client.Collection("links").Doc(id)

	var docSnap *firestore.DocumentSnapshot
	var err error

	if r.tx != nil {
		docSnap, err = r.tx.Get(docRef)
	} else {
		docSnap, err = docRef.Get(ctx)
	}

	if err != nil {
		return nil, err
	}

	var link model.Link
	if err := docSnap.DataTo(&link); err != nil {
		return nil, err
	}
	return &link, nil
}

// FindLinksByUserID
func (r *firestoreLinkRepository) FindLinksByUserID(ctx context.Context, userID string) ([]*model.Link, error) {
	iter := r.client.Collection("links").
		Where("participants", "array-contains", userID).
		Documents(ctx)
	defer iter.Stop()

	var links []*model.Link
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to iterate links: %w", err)
		}

		var link model.Link
		if err := doc.DataTo(&link); err != nil {
			continue
		}
		links = append(links, &link)
	}
	return links, nil
}

// FindLinkByParticipants
func (r *firestoreLinkRepository) FindLinkByParticipants(ctx context.Context, userA, userB string) (*model.Link, error) {
	iter := r.client.Collection("links").
		Where("participants", "array-contains", userA).
		Documents(ctx)
	defer iter.Stop()

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		var link model.Link
		if err := doc.DataTo(&link); err != nil {
			continue
		}

		for _, p := range link.Participants {
			if p == userB {
				return &link, nil
			}
		}
	}

	return nil, nil
}
