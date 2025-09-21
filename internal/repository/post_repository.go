package repository

import (
	"context"
	"github.com/Sayan80bayev/go-project/pkg/logging"
	"time"

	"postService/internal/model"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	createdAtKey = "createdAt"
)

type PostRepositoryImpl struct {
	collection *mongo.Collection
}

func NewPostRepository(db *mongo.Database) *PostRepositoryImpl {
	return &PostRepositoryImpl{
		collection: db.Collection("posts"),
	}
}

func (r *PostRepositoryImpl) CreatePost(ctx context.Context, post *model.Post) error {
	post.ID = uuid.New()
	post.CreatedAt = time.Now().Format(time.RFC3339)
	post.UpdatedAt = post.CreatedAt

	_, err := r.collection.InsertOne(ctx, post)
	return err
}

// GetPosts returns paginated posts
func (r *PostRepositoryImpl) GetPosts(ctx context.Context, page, limit int64) ([]model.Post, error) {
	logger := logging.GetLogger()

	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	skip := (page - 1) * limit

	findOptions := options.Find()
	findOptions.SetSkip(skip)
	findOptions.SetLimit(limit)
	findOptions.SetSort(bson.D{{Key: createdAtKey, Value: -1}}) // newest first

	cursor, err := r.collection.Find(ctx, bson.M{}, findOptions)
	if err != nil {
		return nil, err
	}
	defer func(cursor *mongo.Cursor, ctx context.Context) {
		if cerr := cursor.Close(ctx); cerr != nil {
			logger.Errorf("Error closing cursor: %v", cerr)
		}
	}(cursor, ctx)

	var posts []model.Post
	if err := cursor.All(ctx, &posts); err != nil {
		return nil, err
	}
	return posts, nil
}

// GetPostsByUserID returns paginated posts by userId
func (r *PostRepositoryImpl) GetPostsByUserID(ctx context.Context, userID uuid.UUID, page, limit int64) ([]model.Post, error) {
	logger := logging.GetLogger()

	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	skip := (page - 1) * limit

	findOptions := options.Find()
	findOptions.SetSkip(skip)
	findOptions.SetLimit(limit)
	findOptions.SetSort(bson.D{{Key: createdAtKey, Value: -1}}) // newest first

	filter := bson.M{"userid": userID}

	cursor, err := r.collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, err
	}
	defer func(cursor *mongo.Cursor, ctx context.Context) {
		if cerr := cursor.Close(ctx); cerr != nil {
			logger.Errorf("Error closing cursor: %v", cerr)
		}
	}(cursor, ctx)

	var posts []model.Post
	if err := cursor.All(ctx, &posts); err != nil {
		return nil, err
	}
	return posts, nil
}

func (r *PostRepositoryImpl) GetPostByID(ctx context.Context, id uuid.UUID) (*model.Post, error) {
	var post model.Post
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&post)
	if err != nil {
		return nil, err
	}
	return &post, nil
}

func (r *PostRepositoryImpl) UpdatePost(ctx context.Context, post *model.Post) error {
	post.UpdatedAt = time.Now().Format(time.RFC3339)
	filter := bson.M{"_id": post.ID}
	update := bson.M{"$set": post}
	_, err := r.collection.UpdateOne(ctx, filter, update)
	return err
}

func (r *PostRepositoryImpl) DeletePost(ctx context.Context, id uuid.UUID) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}
