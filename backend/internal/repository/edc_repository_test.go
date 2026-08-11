package repository

import (
	"testing"

	"nfa-dashboard/internal/model"
)

func TestBuildEDCTrafficOrderByOnlyOrdersGroupedDimensions(t *testing.T) {
	tests := []struct {
		name                 string
		filter               model.EDCTrafficFilter
		includeEntityDetails bool
		want                 string
	}{
		{
			name:   "entity type only",
			filter: model.EDCTrafficFilter{EntityType: model.EDCEntityTypeNode},
			want:   "t.bucket_5m ASC, t.entity_type ASC",
		},
		{
			name:   "entity and cp",
			filter: model.EDCTrafficFilter{EntityIDs: []uint64{1}, CP: "ali"},
			want:   "t.bucket_5m ASC, t.cp ASC",
		},
		{
			name:   "source and destination regions",
			filter: model.EDCTrafficFilter{SrcRegion: "北京市", DstRegion: "河北省"},
			want:   "t.bucket_5m ASC, t.src_region ASC, t.dst_region ASC",
		},
		{
			name:                 "entity details",
			includeEntityDetails: true,
			want:                 "t.bucket_5m ASC, t.region ASC, t.cp ASC, t.display_name ASC",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := buildEDCTrafficOrderBy(tt.filter, tt.includeEntityDetails); got != tt.want {
				t.Fatalf("buildEDCTrafficOrderBy() = %q, want %q", got, tt.want)
			}
		})
	}
}
