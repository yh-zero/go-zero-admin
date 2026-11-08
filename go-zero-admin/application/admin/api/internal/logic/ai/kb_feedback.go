package ai

import (
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/ai/rpc/aipb"
)

func mapKbFeedback(row *aipb.KbFeedbackInfo) *types.AiKbFeedbackInfo {
	if row == nil {
		return nil
	}
	var review *types.AiKbFeedbackReview
	if row.Review != nil {
		evidence := make([]*types.AiKbFeedbackEvidence, 0, len(row.Review.Evidence))
		for _, item := range row.Review.Evidence {
			evidence = append(evidence, &types.AiKbFeedbackEvidence{SourceIndex: item.SourceIndex, Quote: item.Quote})
		}
		facts := append([]string{}, row.Review.RequiredFacts...)
		review = &types.AiKbFeedbackReview{Answerable: row.Review.Answerable, ExpectedAnswer: row.Review.ExpectedAnswer, RequiredFacts: facts, Evidence: evidence, Reason: row.Review.Reason}
	}
	return &types.AiKbFeedbackInfo{Id: row.Id, KbId: row.KbId, TaskId: row.TaskId, StepIndex: row.StepIndex, Feedback: row.Feedback, Note: row.Note, Status: row.Status, Revision: row.Revision, Question: row.Question, Answer: row.Answer, Snapshot: row.Snapshot, Review: review, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt, ReviewedAt: row.ReviewedAt}
}

func mapKbFeedbackList(rows []*aipb.KbFeedbackInfo) []*types.AiKbFeedbackInfo {
	list := make([]*types.AiKbFeedbackInfo, 0, len(rows))
	for _, row := range rows {
		list = append(list, mapKbFeedback(row))
	}
	return list
}

func kbFeedbackReviewRPC(review *types.AiKbFeedbackReview) *aipb.KbFeedbackReview {
	if review == nil {
		return nil
	}
	list := make([]*aipb.KbFeedbackEvidence, 0, len(review.Evidence))
	for _, item := range review.Evidence {
		list = append(list, &aipb.KbFeedbackEvidence{SourceIndex: item.SourceIndex, Quote: item.Quote})
	}
	return &aipb.KbFeedbackReview{Answerable: review.Answerable, ExpectedAnswer: review.ExpectedAnswer, RequiredFacts: review.RequiredFacts, Evidence: list, Reason: review.Reason}
}
