// Package lark 提供飞书 SDK Client 的初始化封装。
package lark

import (
	"context"
	"fmt"

	larkbitable "github.com/larksuite/oapi-sdk-go/v3/service/bitable/v1"
	"go.uber.org/zap"

	domain "matrix/api/domain/lark"
)

const bitableMaxBatchSize = 1000

// bitableOperator 实现 domain.BitableOperator。
type bitableOperator struct {
	client *Client
	logger *zap.Logger
}

// compile-time interface check
var _ domain.BitableOperator = (*bitableOperator)(nil)

// NewBitableOperator 创建多维表格操作器。
func NewBitableOperator(client *Client, logger *zap.Logger) domain.BitableOperator {
	return &bitableOperator{client: client, logger: logger}
}

// ListTables 列出多维表格中的所有数据表。
func (b *bitableOperator) ListTables(ctx context.Context, appToken string) ([]domain.BitableTable, error) {
	var result []domain.BitableTable
	var pageToken string

	for {
		req := larkbitable.NewListAppTableReqBuilder().
			AppToken(appToken).
			PageToken(pageToken).
			PageSize(100).
			Build()

		resp, err := b.client.Bitable.AppTable.List(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("bitable: list tables in %s: %w", appToken, err)
		}
		if !resp.Success() {
			return nil, fmt.Errorf("bitable: list tables in %s: code=%d, msg=%s", appToken, resp.Code, resp.Msg)
		}
		if resp.Data == nil {
			break
		}

		for _, t := range resp.Data.Items {
			result = append(result, convertBitableTable(t))
		}

		if resp.Data.HasMore == nil || !*resp.Data.HasMore {
			break
		}
		if resp.Data.PageToken != nil {
			pageToken = *resp.Data.PageToken
		}
	}

	return result, nil
}

// ListFields 获取数据表的字段定义。
func (b *bitableOperator) ListFields(ctx context.Context, appToken string, tableID string) ([]domain.BitableField, error) {
	var result []domain.BitableField
	var pageToken string

	for {
		req := larkbitable.NewListAppTableFieldReqBuilder().
			AppToken(appToken).
			TableId(tableID).
			PageToken(pageToken).
			PageSize(100).
			Build()

		resp, err := b.client.Bitable.AppTableField.List(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("bitable: list fields in %s/%s: %w", appToken, tableID, err)
		}
		if !resp.Success() {
			return nil, fmt.Errorf("bitable: list fields in %s/%s: code=%d, msg=%s", appToken, tableID, resp.Code, resp.Msg)
		}
		if resp.Data == nil {
			break
		}

		for _, f := range resp.Data.Items {
			result = append(result, convertBitableField(f))
		}

		if resp.Data.HasMore == nil || !*resp.Data.HasMore {
			break
		}
		if resp.Data.PageToken != nil {
			pageToken = *resp.Data.PageToken
		}
	}

	return result, nil
}

// SearchRecords 查询记录（支持筛选），处理分页。
func (b *bitableOperator) SearchRecords(ctx context.Context, appToken string, tableID string, filter string) ([]domain.BitableRecord, error) {
	var result []domain.BitableRecord
	var pageToken string

	for {
		bodyBuilder := larkbitable.NewSearchAppTableRecordReqBodyBuilder()
		body := bodyBuilder.Build()

		req := larkbitable.NewSearchAppTableRecordReqBuilder().
			AppToken(appToken).
			TableId(tableID).
			PageToken(pageToken).
			PageSize(500).
			Body(body).
			Build()

		resp, err := b.client.Bitable.AppTableRecord.Search(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("bitable: search records in %s/%s: %w", appToken, tableID, err)
		}
		if !resp.Success() {
			return nil, fmt.Errorf("bitable: search records in %s/%s: code=%d, msg=%s", appToken, tableID, resp.Code, resp.Msg)
		}
		if resp.Data == nil {
			break
		}

		for _, r := range resp.Data.Items {
			result = append(result, convertBitableRecord(r))
		}

		if resp.Data.HasMore == nil || !*resp.Data.HasMore {
			break
		}
		if resp.Data.PageToken != nil {
			pageToken = *resp.Data.PageToken
		}
	}

	return result, nil
}

// BatchCreateRecords 批量新增记录（超过 1000 条自动分批）。
func (b *bitableOperator) BatchCreateRecords(ctx context.Context, appToken string, tableID string, records []domain.BitableRecord) ([]domain.BitableRecord, error) {
	chunks := chunkRecords(records, bitableMaxBatchSize)
	var result []domain.BitableRecord

	for _, chunk := range chunks {
		sdkRecords := domainToSDKRecords(chunk)
		body := larkbitable.NewBatchCreateAppTableRecordReqBodyBuilder().
			Records(sdkRecords).
			Build()

		req := larkbitable.NewBatchCreateAppTableRecordReqBuilder().
			AppToken(appToken).
			TableId(tableID).
			Body(body).
			Build()

		resp, err := b.client.Bitable.AppTableRecord.BatchCreate(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("bitable: batch create records in %s/%s: %w", appToken, tableID, err)
		}
		if !resp.Success() {
			return nil, fmt.Errorf("bitable: batch create records in %s/%s: code=%d, msg=%s", appToken, tableID, resp.Code, resp.Msg)
		}
		if resp.Data != nil {
			for _, r := range resp.Data.Records {
				result = append(result, convertBitableRecord(r))
			}
		}
	}

	return result, nil
}

// BatchUpdateRecords 批量更新记录（超过 1000 条自动分批）。
func (b *bitableOperator) BatchUpdateRecords(ctx context.Context, appToken string, tableID string, records []domain.BitableRecord) ([]domain.BitableRecord, error) {
	chunks := chunkRecords(records, bitableMaxBatchSize)
	var result []domain.BitableRecord

	for _, chunk := range chunks {
		sdkRecords := domainToSDKRecords(chunk)
		body := larkbitable.NewBatchUpdateAppTableRecordReqBodyBuilder().
			Records(sdkRecords).
			Build()

		req := larkbitable.NewBatchUpdateAppTableRecordReqBuilder().
			AppToken(appToken).
			TableId(tableID).
			Body(body).
			Build()

		resp, err := b.client.Bitable.AppTableRecord.BatchUpdate(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("bitable: batch update records in %s/%s: %w", appToken, tableID, err)
		}
		if !resp.Success() {
			return nil, fmt.Errorf("bitable: batch update records in %s/%s: code=%d, msg=%s", appToken, tableID, resp.Code, resp.Msg)
		}
		if resp.Data != nil {
			for _, r := range resp.Data.Records {
				result = append(result, convertBitableRecord(r))
			}
		}
	}

	return result, nil
}

// BatchDeleteRecords 批量删除记录。
func (b *bitableOperator) BatchDeleteRecords(ctx context.Context, appToken string, tableID string, recordIDs []string) error {
	body := larkbitable.NewBatchDeleteAppTableRecordReqBodyBuilder().
		Records(recordIDs).
		Build()

	req := larkbitable.NewBatchDeleteAppTableRecordReqBuilder().
		AppToken(appToken).
		TableId(tableID).
		Body(body).
		Build()

	resp, err := b.client.Bitable.AppTableRecord.BatchDelete(ctx, req)
	if err != nil {
		return fmt.Errorf("bitable: batch delete records in %s/%s: %w", appToken, tableID, err)
	}
	if !resp.Success() {
		return fmt.Errorf("bitable: batch delete records in %s/%s: code=%d, msg=%s", appToken, tableID, resp.Code, resp.Msg)
	}
	return nil
}

// chunkRecords 将记录切分为指定大小的分片。
func chunkRecords(records []domain.BitableRecord, size int) [][]domain.BitableRecord {
	if len(records) == 0 {
		return nil
	}
	var chunks [][]domain.BitableRecord
	for i := 0; i < len(records); i += size {
		end := i + size
		if end > len(records) {
			end = len(records)
		}
		chunks = append(chunks, records[i:end])
	}
	return chunks
}

// convertBitableRecord 将 SDK AppTableRecord 转换为 domain.BitableRecord。
func convertBitableRecord(r *larkbitable.AppTableRecord) domain.BitableRecord {
	if r == nil {
		return domain.BitableRecord{}
	}
	rec := domain.BitableRecord{
		Fields: r.Fields,
	}
	if r.RecordId != nil {
		rec.RecordID = *r.RecordId
	}
	return rec
}

// convertBitableField 将 SDK AppTableFieldForList 转换为 domain.BitableField。
func convertBitableField(f *larkbitable.AppTableFieldForList) domain.BitableField {
	if f == nil {
		return domain.BitableField{}
	}
	field := domain.BitableField{}
	if f.FieldId != nil {
		field.FieldID = *f.FieldId
	}
	if f.FieldName != nil {
		field.FieldName = *f.FieldName
	}
	if f.Type != nil {
		field.FieldType = *f.Type
	}
	return field
}

// convertBitableTable 将 SDK AppTable 转换为 domain.BitableTable。
func convertBitableTable(t *larkbitable.AppTable) domain.BitableTable {
	if t == nil {
		return domain.BitableTable{}
	}
	table := domain.BitableTable{}
	if t.TableId != nil {
		table.TableID = *t.TableId
	}
	if t.Name != nil {
		table.Name = *t.Name
	}
	return table
}

// domainToSDKRecords 将 domain.BitableRecord 切片转换为 SDK AppTableRecord 切片。
func domainToSDKRecords(records []domain.BitableRecord) []*larkbitable.AppTableRecord {
	result := make([]*larkbitable.AppTableRecord, len(records))
	for i, r := range records {
		rec := &larkbitable.AppTableRecord{
			Fields: r.Fields,
		}
		if r.RecordID != "" {
			recID := r.RecordID
			rec.RecordId = &recID
		}
		result[i] = rec
	}
	return result
}
