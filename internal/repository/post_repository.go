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

// NewPostRepository initializes repository and creates indexes if needed
func NewPostRepository(db *mongo.Database) *PostRepositoryImpl {
	collection := db.Collection("posts")

	// Create a compound index on userid and createdAt for efficient filtering + sorting
	indexModel := mongo.IndexModel{
		Keys: bson.D{
			{Key: "userid", Value: 1},
			{Key: createdAtKey, Value: -1},
		},
		Options: options.Index(),
	}
	_, err := collection.Indexes().CreateOne(context.Background(), indexModel)
	if err != nil {
		logging.GetLogger().Warnf("Failed to create index on posts collection: %v", err)
	}

	return &PostRepositoryImpl{
		collection: collection,
	}
}

func (r *PostRepositoryImpl) CreatePost(ctx context.Context, post *model.Post) error {
	post.ID = uuid.New()
	post.CreatedAt = time.Now().Format(time.RFC3339)
	post.UpdatedAt = post.CreatedAt

	_, err := r.collection.InsertOne(ctx, post)
	return err
}

// GetPosts returns paginated posts without filtering by user
func (r *PostRepositoryImpl) GetPosts(ctx context.Context, page, limit int64) (*model.PaginatedPosts, error) {
	return r.getPaginatedPosts(ctx, bson.M{}, page, limit)
}

func (r *PostRepositoryImpl) GetPostsByUserID(ctx context.Context, userID uuid.UUID, page, limit int64) (*model.PaginatedPosts, error) {
	return r.getPaginatedPosts(ctx, bson.M{"userid": userID}, page, limit)
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

// getPaginatedPosts is an internal helper to apply pagination logic on any filter
func (r *PostRepositoryImpl) getPaginatedPosts(ctx context.Context, filter bson.M, page, limit int64) (*model.PaginatedPosts, error) {
	logger := logging.GetLogger()

	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	skip := (page - 1) * limit

	// Counting documents (impact reduced by using index)
	total, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, err
	}

	findOptions := options.Find()
	findOptions.SetSkip(skip)
	findOptions.SetLimit(limit)
	findOptions.SetSort(bson.D{{Key: createdAtKey, Value: -1}})

	cursor, err := r.collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := cursor.Close(ctx); cerr != nil {
			logger.Errorf("Error closing cursor: %v", cerr)
		}
	}()

	var posts []model.Post
	if err := cursor.All(ctx, &posts); err != nil {
		return nil, err
	}

	hasNext := (page * limit) < total

	return &model.PaginatedPosts{
		Posts:   posts,
		Page:    page,
		Limit:   limit,
		Total:   total,
		HasNext: hasNext,
	}, nil
}
