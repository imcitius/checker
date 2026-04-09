// SPDX-License-Identifier: BUSL-1.1

package scheduler

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/imcitius/checker/pkg/db"
	"github.com/imcitius/checker/pkg/models"
	checkersentry "github.com/imcitius/checker/internal/sentry"

	"github.com/sirupsen/logrus"
)

// RunConsensusSweeper periodically evaluates multi-region check results and fires alerts
// based on quorum consensus. It runs as a single goroutine per instance; the atomic
// ClaimCycleForEvaluation ensures only one instance evaluates each cycle.
func RunConsensusSweeper(ctx context.Context, region string, minRegions int, evalInterval, timeout time.Duration, repo db.Repository, appAlerters []AppAlerter) {
	ticker := time.NewTicker(evalInterval)
	defer ticker.Stop()

	purgeCounter := 0

	logrus.Infof("Consensus sweeper started (region=%s, min_regions=%d, eval_interval=%s, timeout=%s)",
		region, minRegions, evalInterval, timeout)

	for {
		select {
		case <-ctx.Done():
			logrus.Info("Consensus sweeper shutting down")
			return
		case <-ticker.C:
			evaluateConsensus(ctx, minRegions, timeout, repo, appAlerters)

			purgeCounter++
			if purgeCounter%100 == 0 {
				n, err := repo.PurgeOldCheckResults(ctx, 24*time.Hour)
				if err != nil {
					logrus.WithError(err).Error("Failed to purge old check results")
				checkersentry.CaptureError(err, map[string]string{"op": "purge_old_check_results"})
				} else if n > 0 {
					logrus.Infof("Purged %d old check results", n)
				}
			}
		}
	}
}

func evaluateConsensus(ctx context.Context, minRegions int, timeout time.Duration, repo db.Repository, appAlerters []AppAlerter) {
	cycles, err := repo.GetUnevaluatedCycles(ctx, minRegions, timeout)
	if err != nil {
		logrus.WithError(err).Error("Failed to get unevaluated cycles")
		checkersentry.CaptureError(err, map[string]string{"op": "get_unevaluated_cycles"})
		return
	}

	for _, cycle := range cycles {
		claimed, err := repo.ClaimCycleForEvaluation(ctx, cycle.CheckUUID, cycle.CycleKey)
		if err != nil {
			logrus.WithError(err).Errorf("Failed to claim cycle for check %s", cycle.CheckUUID)
			continue
		}
		if !claimed {
			continue // another instance got it
		}

		results, err := repo.GetCycleResults(ctx, cycle.CheckUUID, cycle.CycleKey)
		if err != nil {
			logrus.WithError(err).Errorf("Failed to get cycle results for check %s", cycle.CheckUUID)
			continue
		}
		if len(results) == 0 {
			continue
		}

		isHealthy, message, failingRegions := computeConsensus(results)

		checkDef, err := repo.GetCheckDefinitionByUUID(ctx, cycle.CheckUUID)
		if err != nil {
			logrus.WithError(err).Errorf("Failed to load check definition %s for consensus", cycle.CheckUUID)
			continue
		}

		host := ""
		if checkDef.Config != nil {
			host = checkDef.Config.GetTarget()
		}

		checkStatus := models.CheckStatus{
			UUID:          checkDef.UUID,
			Project:       checkDef.Project,
			CheckGroup:    checkDef.GroupName,
			CheckName:     checkDef.Name,
			CheckType:     checkDef.Type,
			LastRun:       cycle.CycleKey,
			IsHealthy:     isHealthy,
			Message:       message,
			IsEnabled:     checkDef.Enabled,
			Host:          host,
			Periodicity:   checkDef.Duration,
			LastAlertSent: checkDef.LastAlertSent,
			Region:        failingRegions,
		}

		logger := logrus.WithFields(logrus.Fields{
			"check":     checkDef.Name,
			"consensus": fmt.Sprintf("%d regions", len(results)),
		})

		processCheckResult(repo, checkDef, checkStatus, isHealthy, cycle.CycleKey, appAlerters, logger)
	}
}

// computeConsensus determines the overall health based on a majority quorum.
// If more than half the reporting regions say unhealthy, the consensus is unhealthy.
// Returns isHealthy, message, and a comma-separated list of failing region names.
func computeConsensus(results []models.CheckResult) (isHealthy bool, message string, failingRegions string) {
	unhealthyCount := 0
	var messages []string
	var regions []string
	for _, r := range results {
		if !r.IsHealthy {
			unhealthyCount++
			messages = append(messages, fmt.Sprintf("[%s] %s", r.Region, r.Message))
			regions = append(regions, r.Region)
		}
	}
	if unhealthyCount > len(results)/2 {
		return false, strings.Join(messages, "; "), strings.Join(regions, ",")
	}
	return true, "", ""
}
