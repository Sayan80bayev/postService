package repository

import (
	"context"
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

func (r *PostRepositoryImpl) CreatePost(post *model.Post) error {
	post.ID = uuid.New()
	post.CreatedAt = time.Now().Format(time.RFC3339)
	post.UpdatedAt = post.CreatedAt

	_, err := r.collection.InsertOne(context.TODO(), post)
	return err
}

func (r *PostRepositoryImpl) GetPosts() ([]model.Post, error) {
	cursor, err := r.collection.Find(context.TODO(), bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())

	var posts []model.Post
	if err := cursor.All(context.TODO(), &posts); err != nil {
		return nil, err
	}
	return posts, nil
}

func (r *PostRepositoryImpl) GetPostByID(id uuid.UUID) (*model.Post, error) {
	var post model.Post
	err := r.collection.FindOne(context.TODO(), bson.M{"_id": id}).Decode(&post)
	if err != nil {
		return nil, err
	}
	return &post, nil
}

func (r *PostRepositoryImpl) UpdatePost(post *model.Post) error {
	post.UpdatedAt = time.Now().Format(time.RFC3339)
	filter := bson.M{"_id": post.ID}
	update := bson.M{"$set": post}
	_, err := r.collection.UpdateOne(context.TODO(), filter, update)
	return err
}

func (r *PostRepositoryImpl) DeletePost(id uuid.UUID) error {
	_, err := r.collection.DeleteOne(context.TODO(), bson.M{"_id": id})
	return err
}
