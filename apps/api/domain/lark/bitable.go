package lark

import "context"

// BitableRecord 多维表格记录。
type BitableRecord struct {
	RecordID string                 // 记录 ID（新增时为空）
	Fields   map[string]interface{} // 字段名 → 值
}

// BitableField 多维表格字段定义。
type BitableField struct {
	FieldID   string // 字段 ID
	FieldName string // 字段名
	FieldType int    // 字段类型
}

// BitableTable 多维表格中的数据表。
type BitableTable struct {
	TableID string // 数据表 ID
	Name    string // 数据表名称
}

// BitableOperator 多维表格通用操作能力。
type BitableOperator interface {
	// ListTables 列出多维表格中的所有数据表。
	ListTables(ctx context.Context, appToken string) ([]BitableTable, error)

	// ListFields 获取数据表的字段定义。
	ListFields(ctx context.Context, appToken string, tableID string) ([]BitableField, error)

	// SearchRecords 查询记录（支持筛选）。
	// filter 为飞书筛选表达式，为空则不筛选。
	SearchRecords(ctx context.Context, appToken string, tableID string, filter string) ([]BitableRecord, error)

	// BatchCreateRecords 批量新增记录（单次最多 1000 条）。
	BatchCreateRecords(ctx context.Context, appToken string, tableID string, records []BitableRecord) ([]BitableRecord, error)

	// BatchUpdateRecords 批量更新记录（单次最多 1000 条）。
	BatchUpdateRecords(ctx context.Context, appToken string, tableID string, records []BitableRecord) ([]BitableRecord, error)

	// BatchDeleteRecords 批量删除记录。
	BatchDeleteRecords(ctx context.Context, appToken string, tableID string, recordIDs []string) error
}
