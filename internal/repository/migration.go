package repository

import (
	"encoding/json"
	"fmt"

	"github.com/pg/dts/internal/model"
	"gorm.io/gorm"
)

// MigrationRepository 迁移任务仓储
type MigrationRepository struct {
	db *gorm.DB
}

// NewMigrationRepository 创建迁移任务仓储
func NewMigrationRepository(db *gorm.DB) *MigrationRepository {
	return &MigrationRepository{db: db}
}

// Create 创建迁移任务
func (r *MigrationRepository) Create(task *model.MigrationTask) error {
	return r.db.Create(task).Error
}

// GetByID 根据ID获取任务
func (r *MigrationRepository) GetByID(id string) (*model.MigrationTask, error) {
	var task model.MigrationTask
	if err := r.db.Where("id = ?", id).First(&task).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

// List 获取任务列表
func (r *MigrationRepository) List(limit, offset int) ([]*model.MigrationTask, error) {
	var tasks []*model.MigrationTask
	err := r.db.Order("created_at DESC").Limit(limit).Offset(offset).Find(&tasks).Error
	return tasks, err
}

// Update 更新任务
func (r *MigrationRepository) Update(task *model.MigrationTask) error {
	return r.db.Save(task).Error
}

// UpdateState 更新任务状态
func (r *MigrationRepository) UpdateState(id string, state model.StateType, errorMsg string) error {
	updates := map[string]interface{}{
		"state": state.String(),
	}

	if errorMsg != "" {
		updates["error_message"] = errorMsg
	}

	return r.db.Model(&model.MigrationTask{}).Where("id = ?", id).Updates(updates).Error
}

// UpdateProgress 更新任务进度
func (r *MigrationRepository) UpdateProgress(id string, progress int) error {
	return r.db.Model(&model.MigrationTask{}).Where("id = ?", id).Update("progress", progress).Error
}

// Delete 删除任务
func (r *MigrationRepository) Delete(id string) error {
	return r.db.Delete(&model.MigrationTask{}, id).Error
}

// ParseSourceDB 解析源数据库配置
func ParseSourceDB(task *model.MigrationTask) (*model.DBConfig, error) {
	var dbConfig model.DBConfig
	if err := json.Unmarshal([]byte(task.SourceDB), &dbConfig); err != nil {
		return nil, fmt.Errorf("failed to parse source db config: %w", err)
	}
	return &dbConfig, nil
}

// ParseTargetDB 解析目标数据库配置
func ParseTargetDB(task *model.MigrationTask) (*model.DBConfig, error) {
	var dbConfig model.DBConfig
	if err := json.Unmarshal([]byte(task.TargetDB), &dbConfig); err != nil {
		return nil, fmt.Errorf("failed to parse target db config: %w", err)
	}
	return &dbConfig, nil
}

// ParseTables 解析表列表
func ParseTables(task *model.MigrationTask) ([]string, error) {
	var tables []string
	if err := json.Unmarshal([]byte(task.Tables), &tables); err != nil {
		return nil, fmt.Errorf("failed to parse tables: %w", err)
	}
	return tables, nil
}
