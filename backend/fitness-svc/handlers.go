package main

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/riefky17/family-app-suite/backend/internal/auth"
)

type fitnessService struct {
	pool *pgxpool.Pool
}

type exercise struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	MuscleGroup string `json:"muscle_group"`
	Equipment   string `json:"equipment"`
}

func (s *fitnessService) listExercises(c *fiber.Ctx) error {
	rows, err := s.pool.Query(c.Context(), `SELECT id, name, muscle_group, equipment FROM fitness.exercises ORDER BY muscle_group, name`)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	defer rows.Close()

	out := []exercise{}
	for rows.Next() {
		var e exercise
		if err := rows.Scan(&e.ID, &e.Name, &e.MuscleGroup, &e.Equipment); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		out = append(out, e)
	}
	return c.JSON(out)
}

type planExerciseInput struct {
	ExerciseID int64    `json:"exercise_id"`
	DayLabel   string   `json:"day_label"`
	Sets       int      `json:"sets"`
	Reps       string   `json:"reps"`
	TargetRPE  *float64 `json:"target_rpe"`
}

type plan struct {
	ID        int64               `json:"id"`
	Name      string              `json:"name"`
	Style     string              `json:"style"`
	CreatedAt string              `json:"created_at"`
	Exercises []planExerciseEntry `json:"exercises,omitempty"`
}

type planExerciseEntry struct {
	ID           int64    `json:"id"`
	ExerciseID   int64    `json:"exercise_id"`
	ExerciseName string   `json:"exercise_name"`
	DayLabel     string   `json:"day_label"`
	Sets         int      `json:"sets"`
	Reps         string   `json:"reps"`
	TargetRPE    *float64 `json:"target_rpe"`
	SortOrder    int      `json:"sort_order"`
}

func (s *fitnessService) listPlans(c *fiber.Ctx) error {
	user := auth.CurrentUser(c)

	rows, err := s.pool.Query(c.Context(), `
		SELECT id, name, style, created_at
		FROM fitness.plans
		WHERE user_id = $1
		ORDER BY created_at DESC
	`, user.ID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	defer rows.Close()

	out := []plan{}
	for rows.Next() {
		var p plan
		if err := rows.Scan(&p.ID, &p.Name, &p.Style, &p.CreatedAt); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		out = append(out, p)
	}
	return c.JSON(out)
}

func (s *fitnessService) getPlan(c *fiber.Ctx) error {
	user := auth.CurrentUser(c)

	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}

	var p plan
	err = s.pool.QueryRow(c.Context(), `
		SELECT id, name, style, created_at FROM fitness.plans WHERE id = $1 AND user_id = $2
	`, id, user.ID).Scan(&p.ID, &p.Name, &p.Style, &p.CreatedAt)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "plan not found")
	}

	rows, err := s.pool.Query(c.Context(), `
		SELECT pe.id, pe.exercise_id, ex.name, pe.day_label, pe.sets, pe.reps, pe.target_rpe, pe.sort_order
		FROM fitness.plan_exercises pe
		JOIN fitness.exercises ex ON ex.id = pe.exercise_id
		WHERE pe.plan_id = $1
		ORDER BY pe.day_label, pe.sort_order
	`, p.ID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	defer rows.Close()

	p.Exercises = []planExerciseEntry{}
	for rows.Next() {
		var pe planExerciseEntry
		if err := rows.Scan(&pe.ID, &pe.ExerciseID, &pe.ExerciseName, &pe.DayLabel, &pe.Sets, &pe.Reps, &pe.TargetRPE, &pe.SortOrder); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		p.Exercises = append(p.Exercises, pe)
	}

	return c.JSON(p)
}

func (s *fitnessService) createPlan(c *fiber.Ctx) error {
	user := auth.CurrentUser(c)

	var body struct {
		Name      string              `json:"name"`
		Style     string              `json:"style"`
		Exercises []planExerciseInput `json:"exercises"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if body.Name == "" {
		return fiber.NewError(fiber.StatusBadRequest, "name is required")
	}
	if body.Style == "" {
		body.Style = "general"
	}

	tx, err := s.pool.Begin(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	defer tx.Rollback(c.Context())

	var planID int64
	err = tx.QueryRow(c.Context(), `
		INSERT INTO fitness.plans (user_id, name, style) VALUES ($1, $2, $3) RETURNING id
	`, user.ID, body.Name, body.Style).Scan(&planID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	for i, pe := range body.Exercises {
		if pe.Sets == 0 {
			pe.Sets = 3
		}
		if pe.Reps == "" {
			pe.Reps = "8-12"
		}
		_, err = tx.Exec(c.Context(), `
			INSERT INTO fitness.plan_exercises (plan_id, exercise_id, day_label, sets, reps, target_rpe, sort_order)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, planID, pe.ExerciseID, pe.DayLabel, pe.Sets, pe.Reps, pe.TargetRPE, i)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
	}

	if err := tx.Commit(c.Context()); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"id": planID})
}

func (s *fitnessService) deletePlan(c *fiber.Ctx) error {
	user := auth.CurrentUser(c)

	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}

	tag, err := s.pool.Exec(c.Context(), `DELETE FROM fitness.plans WHERE id = $1 AND user_id = $2`, id, user.ID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if tag.RowsAffected() == 0 {
		return fiber.NewError(fiber.StatusNotFound, "plan not found")
	}
	return c.JSON(fiber.Map{"ok": true})
}

type session struct {
	ID          int64  `json:"id"`
	PlanID      *int64 `json:"plan_id"`
	Notes       string `json:"notes"`
	PerformedAt string `json:"performed_at"`
}

func (s *fitnessService) listSessions(c *fiber.Ctx) error {
	user := auth.CurrentUser(c)

	rows, err := s.pool.Query(c.Context(), `
		SELECT id, plan_id, coalesce(notes, ''), performed_at
		FROM fitness.sessions
		WHERE user_id = $1
		ORDER BY performed_at DESC
		LIMIT 100
	`, user.ID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	defer rows.Close()

	out := []session{}
	for rows.Next() {
		var sess session
		if err := rows.Scan(&sess.ID, &sess.PlanID, &sess.Notes, &sess.PerformedAt); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		out = append(out, sess)
	}
	return c.JSON(out)
}

func (s *fitnessService) logSession(c *fiber.Ctx) error {
	user := auth.CurrentUser(c)

	var body struct {
		PlanID *int64 `json:"plan_id"`
		Notes  string `json:"notes"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	var id int64
	err := s.pool.QueryRow(c.Context(), `
		INSERT INTO fitness.sessions (user_id, plan_id, notes) VALUES ($1, $2, NULLIF($3, ''))
		RETURNING id
	`, user.ID, body.PlanID, body.Notes).Scan(&id)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"id": id})
}
