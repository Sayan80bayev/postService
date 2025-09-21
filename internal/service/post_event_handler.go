package service

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Sayan80bayev/go-project/pkg/caching"
	"github.com/Sayan80bayev/go-project/pkg/logging"
	storage "github.com/Sayan80bayev/go-project/pkg/objectStorage"
	"postService/internal/cache"
	"postService/internal/events"
	"postService/internal/mappers"
	"postService/internal/repository"
)

var logger = logging.GetLogger()

// PostUpdatedHandler handles post update events
func PostUpdatedHandler(
	cacheService caching.CacheService,
	fileStorage storage.FileStorage,
	postRepo *repository.PostRepositoryImpl,
) func(data json.RawMessage) error {
	return func(data json.RawMessage) error {
		var e events.PostUpdatedEvent
		if err := json.Unmarshal(data, &e); err != nil {
			return fmt.Errorf("failed to unmarshal PostUpdatedEvent: %w", err)
		}

		// using background context (no request scope available here)
		ctx := context.Background()

		// delete old media/files
		for _, oldURL := range e.MediaOldURLs {
			if err := fileStorage.DeleteFileByURL(ctx, oldURL); err != nil {
				logger.Warnf("failed to delete old media file %s: %v", oldURL, err)
			}
		}
		for _, oldURL := range e.FilesOldURLs {
			if err := fileStorage.DeleteFileByURL(ctx, oldURL); err != nil {
				logger.Warnf("failed to delete old file %s: %v", oldURL, err)
			}
		}

		// remove old cache
		if err := cacheService.Delete(ctx, fmt.Sprintf(cache.CacheKeyPostFmt, e.PostID)); err != nil {
			logger.Errorf("failed to delete cache for post %s: %v", e.PostID, err)
		}

		// fetch updated post
		post, err := postRepo.GetPostByID(ctx, e.PostID)
		if err != nil {
			logger.Errorf("failed to fetch post %s by ID: %v", e.PostID, err)
			return nil // don’t retry on fetch errors
		}

		postResponse := mappers.MapPostToResponse(*post)
		jsonData, err := json.Marshal(postResponse)
		if err != nil {
			logger.Warnf("failed to marshal post %s to JSON: %v", e.PostID, err)
			return nil
		}

		// update cache
		if err = cacheService.Set(ctx, fmt.Sprintf(cache.CacheKeyPostFmt, e.PostID), jsonData, cache.CacheTTL); err != nil {
			logger.Errorf("failed to update cache for post %s: %v", e.PostID, err)
			return nil
		}

		logger.Infof("post %s cache updated successfully", e.PostID)
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
		var e events.PostDeletedEvent
		if err := json.Unmarshal(data, &e); err != nil {
			return fmt.Errorf("failed to unmarshal PostDeletedEvent: %w", err)
		}

		ctx := context.Background()

		// delete media/files
		for _, url := range e.MediaURLs {
			if err := fileStorage.DeleteFileByURL(ctx, url); err != nil {
				logger.Warnf("failed to delete media file %s: %v", url, err)
			}
		}
		for _, url := range e.FilesURLs {
			if err := fileStorage.DeleteFileByURL(ctx, url); err != nil {
				logger.Warnf("failed to delete file %s: %v", url, err)
			}
		}

		// remove cache
		if err := cacheService.Delete(ctx, fmt.Sprintf(cache.CacheKeyPostFmt, e.PostID)); err != nil {
			logger.Errorf("failed to delete cache for post %s: %v", e.PostID, err)
		}

		logger.Infof("post %s deleted and cleaned up", e.PostID)

		return nil
	}
}
