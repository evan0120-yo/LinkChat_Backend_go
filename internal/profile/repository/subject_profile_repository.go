package repository

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
	"github.com/evan0120-yo/linkchat-go/internal/profile/model"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type SubjectProfileRepository interface {
	WithTx(tx *firestore.Transaction) SubjectProfileRepository
	SaveSubjectProfile(ctx context.Context, profile *model.SubjectProfile) error
	DeleteSubjectProfile(ctx context.Context, profileID string) error
	FindSubjectProfileByID(ctx context.Context, profileID string) (*model.SubjectProfile, error)
}

type firestoreSubjectProfileRepository struct {
	client *firestore.Client
	tx     *firestore.Transaction
}

func NewSubjectProfileRepository(client *firestore.Client) SubjectProfileRepository {
	return &firestoreSubjectProfileRepository{client: client}
}

func (r *firestoreSubjectProfileRepository) WithTx(tx *firestore.Transaction) SubjectProfileRepository {
	return &firestoreSubjectProfileRepository{
		client: r.client,
		tx:     tx,
	}
}

func (r *firestoreSubjectProfileRepository) SaveSubjectProfile(ctx context.Context, profile *model.SubjectProfile) error {
	docRef := r.client.Collection("subject_profiles").Doc(profile.ID)
	if r.tx != nil {
		return r.tx.Set(docRef, profile)
	}

	_, err := docRef.Set(ctx, profile)
	if err != nil {
		return fmt.Errorf("failed to save subject profile: %w", err)
	}
	return nil
}

func (r *firestoreSubjectProfileRepository) DeleteSubjectProfile(ctx context.Context, profileID string) error {
	docRef := r.client.Collection("subject_profiles").Doc(profileID)
	if r.tx != nil {
		return r.tx.Delete(docRef)
	}

	_, err := docRef.Delete(ctx)
	if err != nil && status.Code(err) != codes.NotFound {
		return fmt.Errorf("failed to delete subject profile: %w", err)
	}
	return nil
}

func (r *firestoreSubjectProfileRepository) FindSubjectProfileByID(ctx context.Context, profileID string) (*model.SubjectProfile, error) {
	docRef := r.client.Collection("subject_profiles").Doc(profileID)

	var (
		docSnap *firestore.DocumentSnapshot
		err     error
	)
	if r.tx != nil {
		docSnap, err = r.tx.Get(docRef)
	} else {
		docSnap, err = docRef.Get(ctx)
	}
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get subject profile: %w", err)
	}

	var profile model.SubjectProfile
	if err := docSnap.DataTo(&profile); err != nil {
		return nil, fmt.Errorf("failed to decode subject profile: %w", err)
	}

	return &profile, nil
}
