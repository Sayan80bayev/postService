package service

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Sayan80bayev/go-project/pkg/caching"
	"postService/internal/cache"
	"postService/internal/events"
	"postService/internal/mappers"
	"postService/internal/repository"
	"time"

	storage "github.com/Sayan80bayev/go-project/pkg/objectStorage"
)

// PostUpdatedHandler handles post update events
func PostUpdatedHandler(
	cacheService caching.CacheService,
	fileStorage storage.FileStorage,
	postRepo *repository.PostRepositoryImpl,
) func(data json.RawMessage) error {
	return func(data json.RawMessage) error {
		var e events.PostUpdated
		if err := json.Unmarshal(data, &e); err != nil {
			return fmt.Errorf("failed to unmarshal PostUpdated: %w", err)
		}

		ctx := context.Background()

		// delete old media/files
		for _, oldURL := range e.MediaOldURLs {
			if err := fileStorage.DeleteFileByURL(oldURL); err != nil {
				logger.Warnf("Failed to delete old file: %s, error: %v", oldURL, err)
			}
		}
		for _, oldURL := range e.FilesOldURLs {
			if err := fileStorage.DeleteFileByURL(oldURL); err != nil {
				logger.Warnf("Failed to delete old file: %s, error: %v", oldURL, err)
			}
		}

		// remove old cache
		if err := cacheService.Delete(ctx, fmt.Sprintf("post:%s", e.PostID)); err != nil {
			logger.Errorf("Error deleting post from cache: %v", err)
		}

		// fetch updated post
		post, err := postRepo.GetPostByID(e.PostID)
		if err != nil {
			logger.Errorf("Error fetching post by ID: %v", err)
			return nil // don’t retry on fetch errors
		}

		postResponse := mappers.MapPostToResponse(*post)
		jsonData, err := json.Marshal(postResponse)
		if err != nil {
			logger.Warnf("Could not marshal post to JSON: %v", err)
			return nil
		}

		// update cache
		if err := cacheService.Set(ctx, fmt.Sprintf("post:%s", e.PostID), jsonData, 5*time.Minute); err != nil {
			logger.Errorf("Failed to update cache: %v", err)
			return nil
		}

		logger.Infof("Post %s cache updated", e.PostID)
		return nil
	}
}

// PostDeletedHandler handles post deletion events
func PostDeletedHandler(
	cacheService caching.CacheService,
	fileStorage storage.FileStorage,
	postRepo *repository.PostRepositoryImpl,
) func(data json.RawMessage) error {
	return func(data json.RawMessage) error {
		var e events.PostDeleted
		if err := json.Unmarshal(data, &e); err != nil {
			return fmt.Errorf("failed to unmarshal PostDeleted: %w", err)
		}

		ctx := context.Background()

		// delete media/files
		for _, url := range e.MediaURLs {
			if err := fileStorage.DeleteFileByURL(url); err != nil {
				logger.Warnf("Error deleting file from MinIO (%s): %v", url, err)
			}
		}
		for _, url := range e.FilesURLs {
			if err := fileStorage.DeleteFileByURL(url); err != nil {
				logger.Warnf("Error deleting file from MinIO (%s): %v", url, err)
			}
		}

		// remove cache
		if err := cacheService.Delete(ctx, fmt.Sprintf("post:%s", e.PostID)); err != nil {
			logger.Errorf("Error deleting post from cache: %v", err)
		}

		logger.Infof("Post %s deleted and cleaned up", e.PostID)

		// refresh global cache
		cache.UpdateCache(cacheService, postRepo)

		return nil
	}
}
