package main

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/riefky17/family-app-suite/backend/internal/auth"
)

type financeService struct {
	pool *pgxpool.Pool
}

type category struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Kind string `json:"kind"`
	Icon string `json:"icon"`
}

func (s *financeService) listCategories(c *fiber.Ctx) error {
	rows, err := s.pool.Query(c.Context(), `SELECT id, name, kind, icon FROM finance.categories ORDER BY kind, name`)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	defer rows.Close()

	var out []category
	for rows.Next() {
		var cat category
		if err := rows.Scan(&cat.ID, &cat.Name, &cat.Kind, &cat.Icon); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		out = append(out, cat)
	}
	return c.JSON(out)
}

type transaction struct {
	ID           int64  `json:"id"`
	UserID       int64  `json:"user_id"`
	CategoryID   int64  `json:"category_id"`
	CategoryName string `json:"category_name"`
	AmountIDR    int64  `json:"amount_idr"`
	Note         string `json:"note"`
	OccurredAt   string `json:"occurred_at"`
}

func (s *financeService) listTransactions(c *fiber.Ctx) error {
	rows, err := s.pool.Query(c.Context(), `
		SELECT t.id, t.user_id, t.category_id, cat.name, t.amount_idr, coalesce(t.note, ''), t.occurred_at
		FROM finance.transactions t
		JOIN finance.categories cat ON cat.id = t.category_id
		ORDER BY t.occurred_at DESC
		LIMIT 200
	`)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	defer rows.Close()

	out := []transaction{}
	for rows.Next() {
		var t transaction
		if err := rows.Scan(&t.ID, &t.UserID, &t.CategoryID, &t.CategoryName, &t.AmountIDR, &t.Note, &t.OccurredAt); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		out = append(out, t)
	}
	return c.JSON(out)
}

func (s *financeService) createTransaction(c *fiber.Ctx) error {
	user := auth.CurrentUser(c)

	var body struct {
		CategoryID int64  `json:"category_id"`
		AmountIDR  int64  `json:"amount_idr"`
		Note       string `json:"note"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if body.CategoryID == 0 || body.AmountIDR == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "category_id and non-zero amount_idr are required")
	}

	var id int64
	err := s.pool.QueryRow(c.Context(), `
		INSERT INTO finance.transactions (user_id, category_id, amount_idr, note)
		VALUES ($1, $2, $3, NULLIF($4, ''))
		RETURNING id
	`, user.ID, body.CategoryID, body.AmountIDR, body.Note).Scan(&id)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"id": id})
}

func (s *financeService) deleteTransaction(c *fiber.Ctx) error {
	user := auth.CurrentUser(c)

	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}

	tag, err := s.pool.Exec(c.Context(),
		`DELETE FROM finance.transactions WHERE id = $1 AND user_id = $2`, id, user.ID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if tag.RowsAffected() == 0 {
		return fiber.NewError(fiber.StatusNotFound, "transaction not found")
	}
	return c.JSON(fiber.Map{"ok": true})
}

func (s *financeService) summary(c *fiber.Ctx) error {
	var incomeIDR, expenseIDR int64
	err := s.pool.QueryRow(c.Context(), `
		SELECT
			coalesce(sum(amount_idr) FILTER (WHERE amount_idr > 0), 0),
			coalesce(sum(-amount_idr) FILTER (WHERE amount_idr < 0), 0)
		FROM finance.transactions
	`).Scan(&incomeIDR, &expenseIDR)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(fiber.Map{
		"income_idr":  incomeIDR,
		"expense_idr": expenseIDR,
		"balance_idr": incomeIDR - expenseIDR,
	})
}
