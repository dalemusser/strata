// internal/app/store/sessions/store.go
package sessions

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Session represents a stored session in the database.
// This is used for server-side session storage if needed.
type Session struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	Token     string             `bson:"token"`
	UserID    primitive.ObjectID `bson:"user_id"`
	Data      map[string]any     `bson:"data,omitempty"`
	ExpiresAt time.Time          `bson:"expires_at"`
	CreatedAt time.Time          `bson:"created_at"`
	UpdatedAt time.Time          `bson:"updated_at"`
}

// Store manages session records in MongoDB.
// Note: Strata primarily uses cookie-based sessions via gorilla/sessions.
// This store is provided for scenarios requiring server-side session storage.
type Store struct {
	c *mongo.Collection
}

// New creates a new session Store.
func New(db *mongo.Database) *Store {
	return &Store{c: db.Collection("sessions")}
}

// EnsureIndexes creates indexes for efficient querying and TTL expiration.
func (s *Store) EnsureIndexes(ctx context.Context) error {
	indexes := []mongo.IndexModel{
		// Lookup by token
		{
			Keys:    bson.D{{Key: "token", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("idx_session_token"),
		},
		// Lookup by user
		{
			Keys:    bson.D{{Key: "user_id", Value: 1}},
			Options: options.Index().SetName("idx_session_user"),
		},
		// TTL index for automatic cleanup
		{
			Keys:    bson.D{{Key: "expires_at", Value: 1}},
			Options: options.Index().SetExpireAfterSeconds(0).SetName("idx_session_ttl"),
		},
	}
	_, err := s.c.Indexes().CreateMany(ctx, indexes)
	return err
}

// Create creates a new session.
func (s *Store) Create(ctx context.Context, session Session) error {
	if session.ID.IsZero() {
		session.ID = primitive.NewObjectID()
	}
	now := time.Now()
	session.CreatedAt = now
	session.UpdatedAt = now
	_, err := s.c.InsertOne(ctx, session)
	return err
}

// GetByToken retrieves a session by token.
func (s *Store) GetByToken(ctx context.Context, token string) (*Session, error) {
	var session Session
	err := s.c.FindOne(ctx, bson.M{
		"token":      token,
		"expires_at": bson.M{"$gt": time.Now()},
	}).Decode(&session)
	if err != nil {
		return nil, err
	}
	return &session, nil
}

// Delete removes a session by token.
func (s *Store) Delete(ctx context.Context, token string) error {
	_, err := s.c.DeleteOne(ctx, bson.M{"token": token})
	return err
}

// DeleteByUser removes all sessions for a user.
func (s *Store) DeleteByUser(ctx context.Context, userID primitive.ObjectID) error {
	_, err := s.c.DeleteMany(ctx, bson.M{"user_id": userID})
	return err
}
