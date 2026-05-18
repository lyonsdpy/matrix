package lark

import (
	"testing"

	larkbitable "github.com/larksuite/oapi-sdk-go/v3/service/bitable/v1"

	domain "matrix/api/domain/lark"
)

func TestChunkRecords(t *testing.T) {
	tests := []struct {
		name       string
		input      []domain.BitableRecord
		chunkSize  int
		wantChunks int
	}{
		{
			name:       "empty slice",
			input:      []domain.BitableRecord{},
			chunkSize:  1000,
			wantChunks: 0,
		},
		{
			name:       "500 records single chunk",
			input:      makeRecords(500),
			chunkSize:  1000,
			wantChunks: 1,
		},
		{
			name:       "exactly 1000 records single chunk",
			input:      makeRecords(1000),
			chunkSize:  1000,
			wantChunks: 1,
		},
		{
			name:       "1001 records two chunks",
			input:      makeRecords(1001),
			chunkSize:  1000,
			wantChunks: 2,
		},
		{
			name:       "1500 records two chunks",
			input:      makeRecords(1500),
			chunkSize:  1000,
			wantChunks: 2,
		},
		{
			name:       "2001 records three chunks",
			input:      makeRecords(2001),
			chunkSize:  1000,
			wantChunks: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunks := chunkRecords(tt.input, tt.chunkSize)
			if len(chunks) != tt.wantChunks {
				t.Errorf("chunkRecords: got %d chunks, want %d", len(chunks), tt.wantChunks)
			}
			// 验证所有 chunk 的总记录数等于输入数量
			total := 0
			for i, chunk := range chunks {
				if len(chunk) > tt.chunkSize {
					t.Errorf("chunk[%d] size %d exceeds limit %d", i, len(chunk), tt.chunkSize)
				}
				total += len(chunk)
			}
			if total != len(tt.input) {
				t.Errorf("total records = %d, want %d", total, len(tt.input))
			}
		})
	}
}

func TestConvertBitableRecord(t *testing.T) {
	tests := []struct {
		name  string
		input *larkbitable.AppTableRecord
		want  domain.BitableRecord
	}{
		{
			name:  "nil record",
			input: nil,
			want:  domain.BitableRecord{},
		},
		{
			name: "record with fields",
			input: &larkbitable.AppTableRecord{
				RecordId: ptrStr("rec001"),
				Fields: map[string]interface{}{
					"姓名": "张三",
					"年龄": float64(30),
				},
			},
			want: domain.BitableRecord{
				RecordID: "rec001",
				Fields: map[string]interface{}{
					"姓名": "张三",
					"年龄": float64(30),
				},
			},
		},
		{
			name: "record without record_id",
			input: &larkbitable.AppTableRecord{
				Fields: map[string]interface{}{
					"key": "value",
				},
			},
			want: domain.BitableRecord{
				RecordID: "",
				Fields: map[string]interface{}{
					"key": "value",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertBitableRecord(tt.input)
			if got.RecordID != tt.want.RecordID {
				t.Errorf("RecordID = %q, want %q", got.RecordID, tt.want.RecordID)
			}
			if len(got.Fields) != len(tt.want.Fields) {
				t.Errorf("Fields len = %d, want %d", len(got.Fields), len(tt.want.Fields))
			}
		})
	}
}

func TestConvertBitableField(t *testing.T) {
	tests := []struct {
		name  string
		input *larkbitable.AppTableFieldForList
		want  domain.BitableField
	}{
		{
			name:  "nil field",
			input: nil,
			want:  domain.BitableField{},
		},
		{
			name: "all fields",
			input: &larkbitable.AppTableFieldForList{
				FieldId:   ptrStr("fld001"),
				FieldName: ptrStr("姓名"),
				Type:      ptrInt(1),
			},
			want: domain.BitableField{
				FieldID:   "fld001",
				FieldName: "姓名",
				FieldType: 1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertBitableField(tt.input)
			if got.FieldID != tt.want.FieldID {
				t.Errorf("FieldID = %q, want %q", got.FieldID, tt.want.FieldID)
			}
			if got.FieldName != tt.want.FieldName {
				t.Errorf("FieldName = %q, want %q", got.FieldName, tt.want.FieldName)
			}
			if got.FieldType != tt.want.FieldType {
				t.Errorf("FieldType = %d, want %d", got.FieldType, tt.want.FieldType)
			}
		})
	}
}

// makeRecords 生成指定数量的测试记录。
func makeRecords(n int) []domain.BitableRecord {
	records := make([]domain.BitableRecord, n)
	for i := range records {
		records[i] = domain.BitableRecord{
			Fields: map[string]interface{}{"index": i},
		}
	}
	return records
}
