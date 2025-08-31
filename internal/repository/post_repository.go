package repository

import (
	"context"
	"github.com/Sayan80bayev/go-project/pkg/logging"
	"time"

	"postService/internal/model"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
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

func (r *PostRepositoryImpl) GetPosts(ctx context.Context) ([]model.Post, error) {
	logger := logging.GetLogger()
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer func(cursor *mongo.Cursor, ctx context.Context) {
		err = cursor.Close(ctx)
		if err != nil {
			logger.Errorf("Error closing cursor: %v", err)
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
