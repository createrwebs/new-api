package model

import (
	"fmt"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupResumableTestDB(t *testing.T) *gorm.DB {
	dsn := fmt.Sprintf("file:test_resumable_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)

	origDB := DB
	t.Cleanup(func() {
		DB = origDB
	})
	DB = db
	require.NoError(t, db.AutoMigrate(
		&StudioWorkflowItem{},
		&ResumableWorkflowJob{},
	))
	return db
}

func TestResumableWorkflowJob_LifecycleAndRecovery(t *testing.T) {
	setupResumableTestDB(t)

	userId := 201
	jobId := "job_batch_resumable_001"

	// 1. Create a 3-item resumable job
	job := &ResumableWorkflowJob{
		Id:         jobId,
		UserId:     userId,
		TotalItems: 3,
	}

	items := []StudioWorkflowItem{
		{Id: "item_0", ItemIndex: 0, OriginalName: "shirt.png", State: ItemStateLocalComplete},
		{Id: "item_1", ItemIndex: 1, OriginalName: "shoes.png", State: ItemStateLocalComplete},
		{Id: "item_2", ItemIndex: 2, OriginalName: "hat.png", State: ItemStatePending},
	}

	err := CreateResumableJob(job, items)
	require.NoError(t, err)
	assert.Equal(t, "IN_PROGRESS", job.Status)

	// 2. Fetch job and items
	fetchedJob, fetchedItems, err := GetResumableJob(userId, jobId)
	require.NoError(t, err)
	assert.Equal(t, 3, len(fetchedItems))
	assert.Equal(t, 3, fetchedJob.TotalItems)
	assert.Equal(t, 0, fetchedJob.CompletedItems)

	// 3. Mark item 0 as SERVER_COMPLETE
	err = UpdateWorkflowItemState(jobId, 0, ItemStateServerComplete, "", "[{\"template\":\"shopee\"}]", 120)
	require.NoError(t, err)

	// 4. Mark item 1 as FAILED_RETRYABLE
	err = UpdateWorkflowItemState(jobId, 1, ItemStateFailedRetryable, "Transient upstream encode error", "", 45)
	require.NoError(t, err)

	// Check job counts updated automatically
	updatedJob, _, err := GetResumableJob(userId, jobId)
	require.NoError(t, err)
	assert.Equal(t, 1, updatedJob.CompletedItems)
	assert.Equal(t, 1, updatedJob.FailedItems)

	// 5. Worker restart / crash simulation: query pending or retryable items
	retryableItems, err := GetPendingOrRetryableItems(jobId)
	require.NoError(t, err)
	assert.Equal(t, 2, len(retryableItems)) // item 1 (retryable) and item 2 (pending)
	assert.Equal(t, 1, retryableItems[0].ItemIndex)
	assert.Equal(t, 2, retryableItems[1].ItemIndex)

	// 6. Retry item 1 to success
	err = UpdateWorkflowItemState(jobId, 1, ItemStateServerComplete, "", "[{\"template\":\"shopee\"}]", 115)
	require.NoError(t, err)

	// Mark item 2 to success
	err = UpdateWorkflowItemState(jobId, 2, ItemStateServerComplete, "", "[{\"template\":\"shopee\"}]", 110)
	require.NoError(t, err)

	// Mark entire job complete
	err = MarkJobCompleted(jobId, "zip_asset_999")
	require.NoError(t, err)

	finalJob, _, err := GetResumableJob(userId, jobId)
	require.NoError(t, err)
	assert.Equal(t, "COMPLETED", finalJob.Status)
	assert.Equal(t, 3, finalJob.CompletedItems)
	assert.Equal(t, 0, finalJob.FailedItems)
	assert.Equal(t, "zip_asset_999", finalJob.ZipAssetId)
}
