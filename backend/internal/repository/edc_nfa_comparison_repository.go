package repository

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
	"nfa-dashboard/internal/model"
)

type EDCNFAComparisonRepository interface {
	ListGroups(ctx context.Context, allowedEntityIDs []uint64) ([]model.EDCNFAComparisonGroup, error)
	GetComparison(ctx context.Context, filter model.EDCNFAComparisonFilter) ([]model.EDCNFAComparisonPoint, error)
}

type EDCNFAComparisonMappingRepository interface {
	EDCNFAComparisonRepository
	ListMappings(ctx context.Context) ([]model.EDCNFAComparisonGroup, error)
	ListMappingEntities(ctx context.Context) ([]model.EDCEntity, error)
	FindActiveMappingMembers(ctx context.Context, entityIDs []uint64, excludeGroupID uint64) ([]model.EDCNFAComparisonGroupMember, error)
	SaveMapping(ctx context.Context, id uint64, input model.EDCNFAComparisonGroupInput) (model.EDCNFAComparisonGroup, error)
	SetMappingEnabled(ctx context.Context, id uint64, enabled bool) error
}

type edcNFAComparisonRepository struct{}

func NewEDCNFAComparisonRepository() EDCNFAComparisonRepository {
	return &edcNFAComparisonRepository{}
}

func (r *edcNFAComparisonRepository) ListMappings(ctx context.Context) ([]model.EDCNFAComparisonGroup, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	var groups []model.EDCNFAComparisonGroup
	if err := model.DB.WithContext(ctx).Order("group_name ASC").Find(&groups).Error; err != nil {
		return nil, err
	}
	if len(groups) == 0 {
		return []model.EDCNFAComparisonGroup{}, nil
	}
	ids := make([]uint64, 0, len(groups))
	for _, group := range groups {
		ids = append(ids, group.ID)
	}
	var members []model.EDCNFAComparisonGroupMember
	err := model.DB.WithContext(ctx).Table("edc_nfa_comparison_group_members AS m").
		Select("m.id, m.group_id, m.entity_id, e.edc_name, e.display_name, m.valid_from, m.valid_to, m.enabled").
		Joins("JOIN edc_entities AS e ON e.id = m.entity_id").
		Where("m.group_id IN ?", ids).
		Order("m.group_id ASC, m.id ASC").Scan(&members).Error
	if err != nil {
		return nil, err
	}
	byGroup := make(map[uint64][]model.EDCNFAComparisonGroupMember, len(groups))
	for _, member := range members {
		byGroup[member.GroupID] = append(byGroup[member.GroupID], member)
	}
	for i := range groups {
		groups[i].Members = byGroup[groups[i].ID]
	}
	return groups, nil
}

func (r *edcNFAComparisonRepository) ListMappingEntities(ctx context.Context) ([]model.EDCEntity, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	var entities []model.EDCEntity
	if err := model.DB.WithContext(ctx).Where("enabled = ? AND is_backup = ?", true, false).
		Order("region ASC, cp ASC, display_name ASC, edc_name ASC").Find(&entities).Error; err != nil {
		return nil, err
	}
	return entities, nil
}

func (r *edcNFAComparisonRepository) FindActiveMappingMembers(ctx context.Context, entityIDs []uint64, excludeGroupID uint64) ([]model.EDCNFAComparisonGroupMember, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if len(entityIDs) == 0 {
		return []model.EDCNFAComparisonGroupMember{}, nil
	}
	query := model.DB.WithContext(ctx).Where("entity_id IN ? AND enabled = ?", entityIDs, true)
	if excludeGroupID > 0 {
		query = query.Where("group_id <> ?", excludeGroupID)
	}
	var members []model.EDCNFAComparisonGroupMember
	if err := query.Find(&members).Error; err != nil {
		return nil, err
	}
	return members, nil
}

func (r *edcNFAComparisonRepository) SaveMapping(ctx context.Context, id uint64, input model.EDCNFAComparisonGroupInput) (model.EDCNFAComparisonGroup, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	var saved model.EDCNFAComparisonGroup
	err := model.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		group := model.EDCNFAComparisonGroup{GroupName: input.GroupName, NFASrcRegion: input.NFASrcRegion, NFACP: input.NFACP, Remark: input.Remark}
		if input.Enabled == nil {
			group.Enabled = true
		} else {
			group.Enabled = *input.Enabled
		}
		if id > 0 {
			if err := tx.First(&group, id).Error; err != nil {
				return err
			}
			group.GroupName = input.GroupName
			group.NFASrcRegion = input.NFASrcRegion
			group.NFACP = input.NFACP
			group.Remark = input.Remark
			if input.Enabled != nil {
				group.Enabled = *input.Enabled
			}
			if err := tx.Save(&group).Error; err != nil {
				return err
			}
		} else if err := tx.Create(&group).Error; err != nil {
			return err
		}
		if err := tx.Where("group_id = ?", group.ID).Delete(&model.EDCNFAComparisonGroupMember{}).Error; err != nil {
			return err
		}
		members := make([]model.EDCNFAComparisonGroupMember, 0, len(input.Members))
		for _, item := range input.Members {
			enabled := true
			if item.Enabled != nil {
				enabled = *item.Enabled
			}
			members = append(members, model.EDCNFAComparisonGroupMember{GroupID: group.ID, EntityID: item.EntityID, ValidFrom: item.ValidFrom, ValidTo: item.ValidTo, Enabled: enabled})
		}
		if len(members) > 0 {
			// EDCName and DisplayName are populated by the read-side join; they are not member-table columns.
			if err := tx.Omit("EDCName", "DisplayName").Create(&members).Error; err != nil {
				return err
			}
		}
		saved = group
		return nil
	})
	if err != nil {
		return model.EDCNFAComparisonGroup{}, err
	}
	groups, err := r.ListMappings(ctx)
	if err != nil {
		return model.EDCNFAComparisonGroup{}, err
	}
	for _, group := range groups {
		if group.ID == saved.ID {
			return group, nil
		}
	}
	return saved, nil
}

func (r *edcNFAComparisonRepository) SetMappingEnabled(ctx context.Context, id uint64, enabled bool) error {
	if ctx == nil {
		ctx = context.Background()
	}
	return model.DB.WithContext(ctx).Model(&model.EDCNFAComparisonGroup{}).Where("id = ?", id).Update("enabled", enabled).Error
}

func (r *edcNFAComparisonRepository) ListGroups(ctx context.Context, allowedEntityIDs []uint64) ([]model.EDCNFAComparisonGroup, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	query := model.DB.WithContext(ctx).Model(&model.EDCNFAComparisonGroup{}).
		Where("enabled = ?", true)
	if len(allowedEntityIDs) > 0 {
		query = query.Where("id IN (?)", model.DB.Table("edc_nfa_comparison_group_members").
			Select("group_id").Where("enabled = ? AND entity_id IN ?", true, allowedEntityIDs))
	}
	var groups []model.EDCNFAComparisonGroup
	if err := query.Order("group_name ASC").Find(&groups).Error; err != nil {
		return nil, err
	}
	if len(groups) == 0 {
		return []model.EDCNFAComparisonGroup{}, nil
	}

	ids := make([]uint64, 0, len(groups))
	for _, group := range groups {
		ids = append(ids, group.ID)
	}
	var members []model.EDCNFAComparisonGroupMember
	memberQuery := model.DB.WithContext(ctx).Table("edc_nfa_comparison_group_members AS m").
		Select("m.id, m.group_id, m.entity_id, e.edc_name, e.display_name, m.valid_from, m.valid_to, m.enabled").
		Joins("JOIN edc_entities AS e ON e.id = m.entity_id AND e.enabled = ? AND e.is_backup = ?", true, false).
		Where("m.enabled = ? AND m.group_id IN ?", true, ids)
	if len(allowedEntityIDs) > 0 {
		memberQuery = memberQuery.Where("m.entity_id IN ?", allowedEntityIDs)
	}
	if err := memberQuery.Order("m.group_id ASC, m.id ASC").Scan(&members).Error; err != nil {
		return nil, err
	}
	byGroup := make(map[uint64][]model.EDCNFAComparisonGroupMember, len(groups))
	for _, member := range members {
		byGroup[member.GroupID] = append(byGroup[member.GroupID], member)
	}
	result := make([]model.EDCNFAComparisonGroup, 0, len(groups))
	for _, group := range groups {
		group.Members = byGroup[group.ID]
		if len(group.Members) == 0 {
			continue
		}
		result = append(result, group)
	}
	return result, nil
}

func (r *edcNFAComparisonRepository) GetComparison(ctx context.Context, filter model.EDCNFAComparisonFilter) ([]model.EDCNFAComparisonPoint, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()

	groups, err := r.ListGroups(ctx, filter.AllowedEntityIDs)
	if err != nil {
		return nil, err
	}
	var group *model.EDCNFAComparisonGroup
	for i := range groups {
		if groups[i].ID == filter.GroupID {
			group = &groups[i]
			break
		}
	}
	if group == nil {
		return []model.EDCNFAComparisonPoint{}, nil
	}
	nfaSchools, err := r.comparisonSchoolsForGroup(ctx, *group, filter.AllowedSchoolKeys)
	if err != nil {
		return nil, err
	}

	// nfa_school_traffic is a raw minute-level fact table. Querying the full
	// window in one GROUP BY causes large temporary tables for multi-day ranges,
	// so read bounded chunks and merge the already-aggregated 5-minute buckets.
	const comparisonChunk = 7 * 24 * time.Hour
	const comparisonChunkConcurrency = 16
	type comparisonChunkRange struct {
		start time.Time
		end   time.Time
	}
	type comparisonChunkResult struct {
		edcRows []comparisonEDCRow
		nfaRows []comparisonNFARow
	}
	chunks := make([]comparisonChunkRange, 0, int(filter.EndTime.Sub(filter.StartTime)/comparisonChunk)+1)
	for cursor := filter.StartTime; cursor.Before(filter.EndTime); {
		chunkEnd := cursor.Add(comparisonChunk)
		if chunkEnd.After(filter.EndTime) {
			chunkEnd = filter.EndTime
		}
		chunks = append(chunks, comparisonChunkRange{start: cursor, end: chunkEnd})
		cursor = chunkEnd
	}

	queryCtx, cancelQueries := context.WithCancel(ctx)
	defer cancelQueries()
	workerCount := comparisonChunkConcurrency
	if len(chunks) < workerCount {
		workerCount = len(chunks)
	}
	jobs := make(chan comparisonChunkRange)
	results := make(chan comparisonChunkResult, len(chunks))
	var workers sync.WaitGroup
	var queryErr error
	var queryErrOnce sync.Once
	recordQueryError := func(err error) {
		queryErrOnce.Do(func() {
			queryErr = err
			cancelQueries()
		})
	}
	for i := 0; i < workerCount; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for chunk := range jobs {
				edcRows, err := r.queryComparisonEDC(queryCtx, *group, chunk.start, chunk.end)
				if err != nil {
					recordQueryError(err)
					return
				}
				nfaRows, err := r.queryComparisonNFA(queryCtx, *group, chunk.start, chunk.end, nfaSchools)
				if err != nil {
					recordQueryError(err)
					return
				}
				results <- comparisonChunkResult{edcRows: edcRows, nfaRows: nfaRows}
			}
		}()
	}

enqueueChunks:
	for _, chunk := range chunks {
		select {
		case jobs <- chunk:
		case <-queryCtx.Done():
			break enqueueChunks
		}
	}
	close(jobs)
	workers.Wait()
	close(results)
	if queryErr != nil {
		return nil, queryErr
	}

	byBucket := make(map[int64]model.EDCNFAComparisonPoint)
	for result := range results {
		edcRows := result.edcRows
		for _, row := range edcRows {
			key := row.Bucket.Unix()
			point := byBucket[key]
			point.Bucket5m = row.Bucket
			point.EDCServiceBytes += row.Bytes
			point.EDCRecordCount += row.RecordCount
			byBucket[key] = point
		}

		nfaRows := result.nfaRows
		for _, row := range nfaRows {
			key := row.Bucket.Unix()
			point := byBucket[key]
			point.Bucket5m = row.Bucket
			point.NFARecvBytes += row.Bytes
			point.NFASchoolCount += row.SchoolCount
			point.NFARecordCount += row.RecordCount
			byBucket[key] = point
		}
	}

	keys := make([]int64, 0, len(byBucket))
	for key := range byBucket {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	points := make([]model.EDCNFAComparisonPoint, 0, len(keys))
	memberSet := make(map[uint64]struct{}, len(group.Members))
	for _, member := range group.Members {
		memberSet[member.EntityID] = struct{}{}
	}
	for _, key := range keys {
		point := byBucket[key]
		point.EDCMemberCount = len(memberSet)
		point.EDCMbps = float64(point.EDCServiceBytes) * 8 / 300 / 1_000_000
		point.NFAMbps = float64(point.NFARecvBytes) * 8 / 60 / 1_000_000
		point.DifferenceMbps = point.NFAMbps - point.EDCMbps
		if point.EDCServiceBytes != 0 {
			ratio := point.NFAMbps / point.EDCMbps
			point.Ratio = &ratio
		}
		switch {
		case point.EDCRecordCount > 0 && point.NFARecordCount > 0:
			point.Status = "ok"
		case point.EDCRecordCount > 0:
			point.Status = "nfa_missing"
		case point.NFARecordCount > 0:
			point.Status = "edc_missing"
		default:
			point.Status = "both_missing"
		}
		points = append(points, point)
	}
	return points, nil
}

type comparisonEDCRow struct {
	Bucket      time.Time
	Bytes       int64
	RecordCount int
}

type comparisonNFARow struct {
	Bucket      time.Time
	Bytes       int64
	SchoolCount int
	RecordCount int
}

func (r *edcNFAComparisonRepository) queryComparisonEDC(ctx context.Context, group model.EDCNFAComparisonGroup, start, end time.Time) ([]comparisonEDCRow, error) {
	var rows []comparisonEDCRow
	memberClauses := make([]string, 0, len(group.Members))
	memberArgs := make([]interface{}, 0, len(group.Members)*3)
	for _, member := range group.Members {
		clause := "t.entity_id = ?"
		memberArgs = append(memberArgs, member.EntityID)
		if member.ValidFrom != nil {
			clause += " AND t.bucket_5m >= ?"
			memberArgs = append(memberArgs, *member.ValidFrom)
		}
		if member.ValidTo != nil {
			clause += " AND t.bucket_5m < ?"
			memberArgs = append(memberArgs, *member.ValidTo)
		}
		memberClauses = append(memberClauses, "("+clause+")")
	}
	if len(memberClauses) == 0 {
		return []comparisonEDCRow{}, nil
	}
	query := model.DB.WithContext(ctx).Table("edc_traffic_5m AS t").
		Select("t.bucket_5m AS bucket, SUM(GREATEST(t.service_size, 0)) AS bytes, COUNT(*) AS record_count").
		Joins("JOIN edc_entities AS e ON e.id = t.entity_id AND e.enabled = ? AND e.is_backup = ?", true, false).
		Where("t.bucket_5m >= ? AND t.bucket_5m < ?", start, end).
		Where(strings.Join(memberClauses, " OR "), memberArgs...)
	if err := query.Group("t.bucket_5m").Order("t.bucket_5m ASC").Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *edcNFAComparisonRepository) comparisonSchoolsForGroup(ctx context.Context, group model.EDCNFAComparisonGroup, allowedSchoolKeys []model.TrafficScopeSchoolKey) ([]model.School, error) {
	var schools []model.School
	if err := model.DB.WithContext(ctx).
		Where("src_region = ? AND cp = ?", group.NFASrcRegion, group.NFACP).
		Order("school_id ASC, region ASC").
		Find(&schools).Error; err != nil {
		return nil, err
	}
	return filterComparisonSchoolsForGroup(schools, allowedSchoolKeys, group.NFASrcRegion, group.NFACP), nil
}

func filterComparisonSchoolsForGroup(schools []model.School, allowedSchoolKeys []model.TrafficScopeSchoolKey, srcRegion, cp string) []model.School {
	if len(allowedSchoolKeys) == 0 {
		return schools
	}
	allowed := make(map[string]struct{}, len(allowedSchoolKeys))
	for _, key := range allowedSchoolKeys {
		if key.SrcRegion == nil || strings.TrimSpace(*key.SrcRegion) != srcRegion || strings.TrimSpace(key.CP) != cp {
			continue
		}
		allowed[comparisonSchoolScopeKey(key.SchoolID, key.Region, key.CP)] = struct{}{}
	}
	filtered := make([]model.School, 0, len(schools))
	for _, school := range schools {
		if school.SrcRegion == nil || strings.TrimSpace(*school.SrcRegion) != srcRegion || strings.TrimSpace(school.CP) != cp {
			continue
		}
		if _, ok := allowed[comparisonSchoolScopeKey(school.SchoolID, school.Region, school.CP)]; ok {
			filtered = append(filtered, school)
		}
	}
	return filtered
}

func comparisonSchoolScopeKey(schoolID, region, cp string) string {
	return strings.TrimSpace(schoolID) + "\x00" + strings.TrimSpace(region) + "\x00" + strings.TrimSpace(cp)
}

func comparisonNFASchoolClauses(schools []model.School) (string, []interface{}) {
	clauses := make([]string, 0, len(schools))
	args := make([]interface{}, 0, len(schools)*3)
	for _, school := range schools {
		schoolID := strings.TrimSpace(school.SchoolID)
		region := strings.TrimSpace(school.Region)
		schoolName := strings.TrimSpace(school.SchoolName)
		if schoolID == "" || region == "" || schoolName == "" {
			continue
		}
		clauses = append(clauses, "(s.school_id = ? AND s.region = ? AND s.school_name = ?)")
		args = append(args, schoolID, region, schoolName)
	}
	if len(clauses) == 0 {
		return "", nil
	}
	return "(" + strings.Join(clauses, " OR ") + ")", args
}

func (r *edcNFAComparisonRepository) queryComparisonNFA(ctx context.Context, group model.EDCNFAComparisonGroup, start, end time.Time, schools []model.School) ([]comparisonNFARow, error) {
	clause, args := comparisonNFASchoolClauses(schools)
	if clause == "" {
		return []comparisonNFARow{}, nil
	}
	bucketExpr := "FROM_UNIXTIME(UNIX_TIMESTAMP(s.create_time) - MOD(UNIX_TIMESTAMP(s.create_time), 300))"
	// Match exact school keys so MySQL can use the existing region/CP/name/time covering index.
	query := model.DB.WithContext(ctx).Table("nfa_school_traffic AS s FORCE INDEX (idx_traffic_rcn_name_time_cov)").
		Select(bucketExpr+" AS bucket, SUM(GREATEST(s.total_recv, 0)) AS bytes, COUNT(DISTINCT s.school_id) AS school_count, COUNT(*) AS record_count").
		Where("s.create_time >= ? AND s.create_time < ? AND s.cp = ?", start, end, group.NFACP).
		Where(clause, args...)
	var rows []comparisonNFARow
	if err := query.Group(bucketExpr).Order("bucket ASC").Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func applyComparisonSchoolScope(query *gorm.DB, keys []model.TrafficScopeSchoolKey) *gorm.DB {
	clause, args := comparisonSchoolScopeClauses(keys)
	if clause == "" {
		return query
	}
	return query.Where(clause, args...)
}

func comparisonSchoolScopeClauses(keys []model.TrafficScopeSchoolKey) (string, []interface{}) {
	if len(keys) == 0 {
		return "", nil
	}
	clauses := make([]string, 0, len(keys))
	args := make([]interface{}, 0, len(keys)*3)
	for _, key := range keys {
		clauses = append(clauses, "(s.school_id = ? AND s.region = ? AND s.cp = ?)")
		args = append(args, key.SchoolID, key.Region, key.CP)
	}
	return "(" + strings.Join(clauses, " OR ") + ")", args
}
