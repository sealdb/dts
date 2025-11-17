package repository

import (
	"encoding/json"
	"fmt"

	"github.com/pg/dts/internal/model"
	"gorm.io/gorm"
)

// MigrationRepository manages migration tasks
type MigrationRepository struct {
	db *gorm.DB
}

// NewMigrationRepository creates a migration task repository
func NewMigrationRepository(db *gorm.DB) *MigrationRepository {
	return &MigrationRepository{db: db}
}

// Create creates a migration task
func (r *MigrationRepository) Create(task *model.MigrationTask) error {
	return r.db.Create(task).Error
}

// GetByID gets task by ID
func (r *MigrationRepository) GetByID(id string) (*model.MigrationTask, error) {
	var task model.MigrationTask
	if err := r.db.Where("id = ?", id).First(&task).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

// List gets task list
func (r *MigrationRepository) List(limit, offset int) ([]*model.MigrationTask, error) {
	var tasks []*model.MigrationTask
	err := r.db.Order("created_at DESC").Limit(limit).Offset(offset).Find(&tasks).Error
	return tasks, err
}

// Update updates a task
func (r *MigrationRepository) Update(task *model.MigrationTask) error {
	return r.db.Save(task).Error
}

// UpdateState updates task state
func (r *MigrationRepository) UpdateState(id string, state model.StateType, errorMsg string) error {
	updates := map[string]interface{}{
		"state": state.String(),
	}

	if errorMsg != "" {
		updates["error_message"] = errorMsg
	}

	return r.db.Model(&model.MigrationTask{}).Where("id = ?", id).Updates(updates).Error
}

// UpdateProgress updates task progress
func (r *MigrationRepository) UpdateProgress(id string, progress int) error {
	return r.db.Model(&model.MigrationTask{}).Where("id = ?", id).Update("progress", progress).Error
}

// Delete deletes a task
func (r *MigrationRepository) Delete(id string) error {
	return r.db.Delete(&model.MigrationTask{}, id).Error
}

// ParseSourceDB parses source database configuration
func ParseSourceDB(task *model.MigrationTask) (*model.DBConfig, error) {
	var dbConfig model.DBConfig
	if err := json.Unmarshal([]byte(task.SourceDB), &dbConfig); err != nil {
		return nil, fmt.Errorf("failed to parse source db config: %w", err)
	}
	return &dbConfig, nil
}

// ParseTargetDB parses target database configuration
func ParseTargetDB(task *model.MigrationTask) (*model.DBConfig, error) {
	var dbConfig model.DBConfig
	if err := json.Unmarshal([]byte(task.TargetDB), &dbConfig); err != nil {
		return nil, fmt.Errorf("failed to parse target db config: %w", err)
	}
	return &dbConfig, nil
}

// ParseTables parses table list
func ParseTables(task *model.MigrationTask) ([]string, error) {
	var tables []string
	if err := json.Unmarshal([]byte(task.Tables), &tables); err != nil {
		return nil, fmt.Errorf("failed to parse tables: %w", err)
	}
	return tables, nil
}
