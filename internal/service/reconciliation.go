package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/example/aigateway/internal/domain"
	"github.com/example/aigateway/pkg/logger"
	"github.com/example/aigateway/pkg/utils"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type ReconciliationService struct {
	db *gorm.DB
}

func NewReconciliationService(db *gorm.DB) *ReconciliationService {
	return &ReconciliationService{db: db}
}

func (s *ReconciliationService) RunAll(ctx context.Context) error {
	logger.L.Info("starting daily reconciliation")

	types := []string{"token_quota", "usage_stats", "cost"}
	for _, t := range types {
		if err := s.Run(ctx, t); err != nil {
			logger.L.Error("reconciliation failed", zap.String("type", t), zap.Error(err))
		}
	}

	logger.L.Info("daily reconciliation completed")
	return nil
}

func (s *ReconciliationService) Run(ctx context.Context, reconciliationType string) error {
	switch reconciliationType {
	case "token_quota":
		return s.reconcileTokenQuota(ctx)
	case "usage_stats":
		return s.reconcileUsageStats(ctx)
	case "cost":
		return s.reconcileCost(ctx)
	default:
		return fmt.Errorf("unknown reconciliation type: %s", reconciliationType)
	}
}

func (s *ReconciliationService) reconcileTokenQuota(ctx context.Context) error {
	now := time.Now()

	type tokenStat struct {
		TokenID           string
		ExpectedRemaining int64
		ActualRemaining   int64
	}

	var results []tokenStat
	rows, err := s.db.WithContext(ctx).Raw(`
		SELECT
			t.id AS token_id,
			t.quota_remaining AS expected_remaining,
			COALESCE(t.quota_total - COALESCE(SUM(rl.input_tokens + rl.output_tokens), 0), t.quota_remaining) AS actual_remaining
		FROM tokens t
		LEFT JOIN request_logs rl ON t.id = rl.token_id
		WHERE t.quota_total IS NOT NULL AND t.quota_total > 0
		GROUP BY t.id
	`).Rows()
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var rs tokenStat
		if err := rows.Scan(&rs.TokenID, &rs.ExpectedRemaining, &rs.ActualRemaining); err != nil {
			continue
		}
		results = append(results, rs)
	}

	for _, r := range results {
		difference := r.ExpectedRemaining - r.ActualRemaining
		status := "matched"
		if difference > 10 || difference < -10 {
			status = "mismatched"
		}

		beforeJSON, _ := json.Marshal(map[string]int64{"expected_remaining": r.ExpectedRemaining})
		afterJSON, _ := json.Marshal(map[string]int64{"actual_remaining": r.ActualRemaining})
		diffJSON, _ := json.Marshal(map[string]int64{"difference": difference})

		record := domain.ReconciliationRecord{
			ID:          utils.GenerateRequestID(),
			Type:        "token_quota",
			Status:      status,
			BeforeState: string(beforeJSON),
			AfterState:  string(afterJSON),
			Difference:  string(diffJSON),
			CorrectedAt: &now,
			Notes:       fmt.Sprintf("token_id: %s", r.TokenID),
		}

		if err := s.db.WithContext(ctx).Create(&record).Error; err != nil {
			logger.L.Error("failed to create reconciliation record", zap.Error(err))
		}
	}

	return nil
}

func (s *ReconciliationService) reconcileUsageStats(ctx context.Context) error {
	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)

	var totalTokens int64
	s.db.WithContext(ctx).Model(&domain.RequestLog{}).
		Where("timestamp >= ? AND timestamp < ?", yesterday, now).
		Select("COALESCE(SUM(input_tokens + output_tokens), 0)").
		Scan(&totalTokens)

	beforeJSON, _ := json.Marshal(map[string]int64{"total_tokens": totalTokens})

	record := domain.ReconciliationRecord{
		ID:          utils.GenerateRequestID(),
		Type:        "usage_stats",
		Status:      "matched",
		BeforeState: string(beforeJSON),
		AfterState:  "{}",
		Difference:  "{}",
		CorrectedAt: &now,
		Notes:       "daily usage stats aggregation",
	}

	return s.db.WithContext(ctx).Create(&record).Error
}

func (s *ReconciliationService) reconcileCost(ctx context.Context) error {
	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)

	var totalCost float64
	s.db.WithContext(ctx).Model(&domain.RequestLog{}).
		Where("timestamp >= ? AND timestamp < ?", yesterday, now).
		Select("COALESCE(SUM(estimated_cost), 0)").
		Scan(&totalCost)

	beforeJSON, _ := json.Marshal(map[string]float64{"total_cost": totalCost})

	record := domain.ReconciliationRecord{
		ID:          utils.GenerateRequestID(),
		Type:        "cost",
		Status:      "matched",
		BeforeState: string(beforeJSON),
		AfterState:  "{}",
		Difference:  "{}",
		CorrectedAt: &now,
		Notes:       "daily cost aggregation",
	}

	return s.db.WithContext(ctx).Create(&record).Error
}
