package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/renos-project/renos/core/internal/db"
	"github.com/renos-project/renos/core/internal/notify"
)

// --- Workflows ---

func listWorkflowsHandler(workflows *db.WorkflowRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		list, err := workflows.ListAll(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to list workflows: "+err.Error())
			return
		}
		if list == nil {
			list = []*db.Workflow{}
		}
		writeJSON(w, http.StatusOK, list)
	}
}

func getWorkflowHandler(workflows *db.WorkflowRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		wf, err := workflows.FindByID(r.Context(), id)
		if err != nil {
			writeError(w, http.StatusNotFound, "not_found", "workflow not found")
			return
		}
		writeJSON(w, http.StatusOK, wf)
	}
}

func createWorkflowHandler(workflows *db.WorkflowRepository, audits *db.AuditRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var wf db.Workflow
		if err := json.NewDecoder(r.Body).Decode(&wf); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
			return
		}
		if wf.Name == "" {
			writeError(w, http.StatusBadRequest, "invalid_request", "name is required")
			return
		}
		if wf.TriggerType == "" {
			wf.TriggerType = "manual"
		}
		wf.Enabled = true
		if wf.TriggerConfig == nil {
			wf.TriggerConfig = json.RawMessage(`{}`)
		}
		if wf.Tags == nil {
			wf.Tags = json.RawMessage(`[]`)
		}
		uid := getUserID(r)
		wf.CreatedBy = &uid
		if err := workflows.Create(r.Context(), &wf); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to create workflow: "+err.Error())
			return
		}
		audits.Create(r.Context(), "user", uid, "workflow.create", "workflow", wf.ID)
		writeJSON(w, http.StatusCreated, wf)
	}
}

func updateWorkflowHandler(workflows *db.WorkflowRepository, audits *db.AuditRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		existing, err := workflows.FindByID(r.Context(), id)
		if err != nil {
			writeError(w, http.StatusNotFound, "not_found", "workflow not found")
			return
		}
		var update db.Workflow
		if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
			return
		}
		if update.Name != "" {
			existing.Name = update.Name
		}
		if update.Description != "" {
			existing.Description = update.Description
		}
		if update.TriggerType != "" {
			existing.TriggerType = update.TriggerType
		}
		if update.TriggerConfig != nil {
			existing.TriggerConfig = update.TriggerConfig
		}
		if update.Tags != nil {
			existing.Tags = update.Tags
		}
		existing.Enabled = update.Enabled
		if err := workflows.Update(r.Context(), existing); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to update workflow: "+err.Error())
			return
		}
		audits.Create(r.Context(), "user", getUserID(r), "workflow.update", "workflow", id)
		writeJSON(w, http.StatusOK, existing)
	}
}

func deleteWorkflowHandler(workflows *db.WorkflowRepository, stepRepos *db.WorkflowStepRepository, audits *db.AuditRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		_ = stepRepos.DeleteByWorkflow(r.Context(), id)
		if err := workflows.Delete(r.Context(), id); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to delete workflow: "+err.Error())
			return
		}
		audits.Create(r.Context(), "user", getUserID(r), "workflow.delete", "workflow", id)
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
	}
}

func toggleWorkflowHandler(workflows *db.WorkflowRepository, audits *db.AuditRepository) http.HandlerFunc {
	type request struct {
		Enabled bool `json:"enabled"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
			return
		}
		if err := workflows.ToggleEnabled(r.Context(), id, req.Enabled); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to toggle workflow: "+err.Error())
			return
		}
		audits.Create(r.Context(), "user", getUserID(r), "workflow.toggle", "workflow", id)
		writeJSON(w, http.StatusOK, map[string]bool{"enabled": req.Enabled})
	}
}

// --- Steps ---

func listWorkflowStepsHandler(steps *db.WorkflowStepRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		list, err := steps.ListByWorkflow(r.Context(), id)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to list steps: "+err.Error())
			return
		}
		if list == nil {
			list = []*db.WorkflowStep{}
		}
		writeJSON(w, http.StatusOK, list)
	}
}

func createWorkflowStepHandler(steps *db.WorkflowStepRepository, audits *db.AuditRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		var step db.WorkflowStep
		if err := json.NewDecoder(r.Body).Decode(&step); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
			return
		}
		if step.Name == "" {
			writeError(w, http.StatusBadRequest, "invalid_request", "name is required")
			return
		}
		if step.StepType == "" {
			step.StepType = "action"
		}
		step.WorkflowID = id
		if step.Config == nil {
			step.Config = json.RawMessage(`{}`)
		}
		if step.ErrorHandler == nil {
			step.ErrorHandler = json.RawMessage(`{}`)
		}
		if err := steps.Create(r.Context(), &step); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to create step: "+err.Error())
			return
		}
		audits.Create(r.Context(), "user", getUserID(r), "workflow.step.create", "workflow", id)
		writeJSON(w, http.StatusCreated, step)
	}
}

func deleteWorkflowStepsHandler(steps *db.WorkflowStepRepository, audits *db.AuditRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if err := steps.DeleteByWorkflow(r.Context(), id); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to delete steps: "+err.Error())
			return
		}
		audits.Create(r.Context(), "user", getUserID(r), "workflow.steps.delete", "workflow", id)
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
	}
}

// --- Runs ---

func listWorkflowRunsHandler(runs *db.WorkflowRunRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		list, err := runs.ListByWorkflow(r.Context(), id, 50)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to list runs: "+err.Error())
			return
		}
		if list == nil {
			list = []*db.WorkflowRun{}
		}
		writeJSON(w, http.StatusOK, list)
	}
}

func getWorkflowRunHandler(runs *db.WorkflowRunRepository, stepRuns *db.WorkflowStepRunRepository) http.HandlerFunc {
	type runDetail struct {
		*db.WorkflowRun
		StepRuns []*db.WorkflowStepRun `json:"step_runs"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		runID := chi.URLParam(r, "runId")
		run, err := runs.FindByID(r.Context(), runID)
		if err != nil {
			writeError(w, http.StatusNotFound, "not_found", "run not found")
			return
		}
		srs, _ := stepRuns.ListByRun(r.Context(), runID)
		if srs == nil {
			srs = []*db.WorkflowStepRun{}
		}
		writeJSON(w, http.StatusOK, runDetail{WorkflowRun: run, StepRuns: srs})
	}
}

func executeWorkflowHandler(workflows *db.WorkflowRepository, runs *db.WorkflowRunRepository, stepRepos *db.WorkflowStepRepository, stepRuns *db.WorkflowStepRunRepository, audits *db.AuditRepository, notifier *notify.Notifier) http.HandlerFunc {
	type request struct {
		TriggerData json.RawMessage `json:"trigger_data"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		wf, err := workflows.FindByID(r.Context(), id)
		if err != nil {
			writeError(w, http.StatusNotFound, "not_found", "workflow not found")
			return
		}
		if !wf.Enabled {
			writeError(w, http.StatusBadRequest, "workflow_disabled", "workflow is disabled")
			return
		}

		var req request
		json.NewDecoder(r.Body).Decode(&req)
		if req.TriggerData == nil {
			req.TriggerData = json.RawMessage(`{}`)
		}

		run := &db.WorkflowRun{
			WorkflowID:  id,
			Status:      "pending",
			TriggerType: "manual",
			TriggerData: req.TriggerData,
			Context:     json.RawMessage(`{}`),
		}
		if err := runs.Create(r.Context(), run); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to create run: "+err.Error())
			return
		}

		// Get steps and create step runs
		steps, _ := stepRepos.ListByWorkflow(r.Context(), id)
		for _, step := range steps {
			sr := &db.WorkflowStepRun{
				RunID:  run.ID,
				StepID: step.ID,
				Status: "pending",
				Input:  json.RawMessage(`{}`),
			}
			stepRuns.Create(r.Context(), sr)
		}

		// Start execution asynchronously
		go executeWorkflowRun(context.Background(), wf, run, steps, runs, stepRuns, stepRepos, notifier)

		audits.Create(r.Context(), "user", getUserID(r), "workflow.execute", "workflow", id)
		writeJSON(w, http.StatusCreated, run)
	}
}

func executeWorkflowRun(ctx context.Context, wf *db.Workflow, run *db.WorkflowRun, steps []*db.WorkflowStep, runs *db.WorkflowRunRepository, stepRuns *db.WorkflowStepRunRepository, stepRepos *db.WorkflowStepRepository, notifier *notify.Notifier) {
	runs.Start(ctx, run.ID)

	for _, step := range steps {
		runs.SetCurrentStep(ctx, run.ID, step.ID)

		srs, _ := stepRuns.ListByRun(ctx, run.ID)
		var currentSR *db.WorkflowStepRun
		for _, sr := range srs {
			if sr.StepID == step.ID {
				currentSR = sr
				break
			}
		}
		if currentSR == nil {
			continue
		}

		stepRuns.Start(ctx, currentSR.ID)

		switch step.StepType {
		case "action":
			// Simulate action execution
			stepRuns.Complete(ctx, currentSR.ID, "completed", `{"result": "executed"}`, "")
		case "condition":
			stepRuns.Complete(ctx, currentSR.ID, "completed", `{"result": true}`, "")
		case "delay":
			stepRuns.Complete(ctx, currentSR.ID, "completed", `{"result": "delayed"}`, "")
		default:
			stepRuns.Complete(ctx, currentSR.ID, "completed", `{}`, "")
		}
	}

	runs.Complete(ctx, run.ID, "completed", "")
	if notifier != nil {
		var creator string
		if wf != nil && wf.CreatedBy != nil {
			creator = *wf.CreatedBy
		}
		notifier.Notify(ctx, creator, "workflow", "Workflow completed",
			"Workflow “"+wf.Name+"” finished successfully.", "workflow", run.ID,
			map[string]any{"workflow_id": wf.ID, "run_id": run.ID})
	}
}

// --- Templates ---

func listWorkflowTemplatesHandler(templates *db.WorkflowTemplateRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		list, err := templates.ListAll(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to list templates: "+err.Error())
			return
		}
		if list == nil {
			list = []*db.WorkflowTemplate{}
		}
		writeJSON(w, http.StatusOK, list)
	}
}

func getWorkflowTemplateHandler(templates *db.WorkflowTemplateRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		t, err := templates.FindByID(r.Context(), id)
		if err != nil {
			writeError(w, http.StatusNotFound, "not_found", "template not found")
			return
		}
		writeJSON(w, http.StatusOK, t)
	}
}

func createWorkflowFromTemplateHandler(workflows *db.WorkflowRepository, steps *db.WorkflowStepRepository, templates *db.WorkflowTemplateRepository, audits *db.AuditRepository) http.HandlerFunc {
	type templateStep struct {
		Name     string          `json:"name"`
		StepType string          `json:"step_type"`
		Config   json.RawMessage `json:"config"`
	}
	type templateDef struct {
		Name          string           `json:"name"`
		Description   string           `json:"description"`
		TriggerType   string           `json:"trigger_type"`
		TriggerConfig json.RawMessage  `json:"trigger_config"`
		Steps         []templateStep   `json:"steps"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		t, err := templates.FindByID(r.Context(), id)
		if err != nil {
			writeError(w, http.StatusNotFound, "not_found", "template not found")
			return
		}

		var td templateDef
		json.Unmarshal(t.Steps, &td.Steps)
		td.Name = t.Name
		td.Description = t.Description
		td.TriggerType = t.TriggerType
		td.TriggerConfig = t.TriggerConfig

		wf := &db.Workflow{
			Name:          t.Name,
			Description:   t.Description,
			Enabled:       true,
			TriggerType:   t.TriggerType,
			TriggerConfig: t.TriggerConfig,
			Tags:          json.RawMessage(`[]`),
		}
		uid := getUserID(r)
		wf.CreatedBy = &uid

		if err := workflows.Create(r.Context(), wf); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to create workflow: "+err.Error())
			return
		}

		for i, ts := range td.Steps {
			step := &db.WorkflowStep{
				WorkflowID:   wf.ID,
				StepOrder:    i,
				Name:         ts.Name,
				StepType:     ts.StepType,
				Config:       ts.Config,
				ErrorHandler: json.RawMessage(`{}`),
			}
			if step.Config == nil {
				step.Config = json.RawMessage(`{}`)
			}
			steps.Create(r.Context(), step)
		}

		audits.Create(r.Context(), "user", uid, "workflow.createFromTemplate", "workflow", wf.ID)
		writeJSON(w, http.StatusCreated, wf)
	}
}

// --- All Runs ---

func listAllWorkflowRunsHandler(runs *db.WorkflowRunRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		list, err := runs.ListAll(r.Context(), 100)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to list runs: "+err.Error())
			return
		}
		if list == nil {
			list = []*db.WorkflowRun{}
		}
		writeJSON(w, http.StatusOK, list)
	}
}
