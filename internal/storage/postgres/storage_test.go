package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Fa1ry7a1l/go-first-proj/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestCreateUser(t *testing.T) {
	db := &fakeDB{
		row: fakeRow{values: []any{int64(7), "user", "hash", time.Unix(10, 0)}},
	}
	storage := &Storage{pool: db}

	user, err := storage.CreateUser(context.Background(), domain.User{Login: "user", PasswordHash: "hash"})
	if err != nil {
		t.Fatalf("CreateUser returned error: %v", err)
	}
	if user.ID != 7 || user.Login != "user" {
		t.Fatalf("user = %+v, want id 7 and login user", user)
	}
}

func TestCreateUserConflict(t *testing.T) {
	db := &fakeDB{
		row: fakeRow{err: &pgconn.PgError{Code: uniqueViolationCode}},
	}
	storage := &Storage{pool: db}

	_, err := storage.CreateUser(context.Background(), domain.User{Login: "user", PasswordHash: "hash"})
	if !errors.Is(err, domain.ErrUserAlreadyExists) {
		t.Fatalf("CreateUser error = %v, want %v", err, domain.ErrUserAlreadyExists)
	}
}

func TestGetUserByLoginNotFound(t *testing.T) {
	storage := &Storage{pool: &fakeDB{row: fakeRow{err: pgx.ErrNoRows}}}

	_, err := storage.GetUserByLogin(context.Background(), "missing")
	if !errors.Is(err, domain.ErrUserNotFound) {
		t.Fatalf("GetUserByLogin error = %v, want %v", err, domain.ErrUserNotFound)
	}
}

func TestCreateOrder(t *testing.T) {
	db := &fakeDB{}
	storage := &Storage{pool: db}

	err := storage.CreateOrder(context.Background(), domain.Order{
		UserID: 1,
		Number: "12345678903",
		Status: domain.OrderStatusNew,
	})
	if err != nil {
		t.Fatalf("CreateOrder returned error: %v", err)
	}
	if len(db.execs) != 1 {
		t.Fatalf("execs = %d, want 1", len(db.execs))
	}
}

func TestCreateOrderConflict(t *testing.T) {
	db := &fakeDB{execErr: &pgconn.PgError{Code: uniqueViolationCode}}
	storage := &Storage{pool: db}

	err := storage.CreateOrder(context.Background(), domain.Order{Number: "12345678903"})
	if !errors.Is(err, domain.ErrOrderUploadedByAnotherUser) {
		t.Fatalf("CreateOrder error = %v, want %v", err, domain.ErrOrderUploadedByAnotherUser)
	}
}

func TestGetOrderByNumber(t *testing.T) {
	points := int64(1234)
	now := time.Unix(10, 0)
	storage := &Storage{pool: &fakeDB{row: fakeRow{values: []any{
		int64(1), "12345678903", int64(2), domain.OrderStatusProcessed, &points, now, now,
	}}}}

	order, err := storage.GetOrderByNumber(context.Background(), "12345678903")
	if err != nil {
		t.Fatalf("GetOrderByNumber returned error: %v", err)
	}
	if order.Accrual == nil || *order.Accrual != domain.Points(points) {
		t.Fatalf("accrual = %v, want %d", order.Accrual, points)
	}
}

func TestListUserOrders(t *testing.T) {
	now := time.Unix(10, 0)
	storage := &Storage{pool: &fakeDB{rows: &fakeRows{rows: [][]any{
		{int64(1), "12345678903", int64(2), domain.OrderStatusNew, (*int64)(nil), now, now},
	}}}}

	orders, err := storage.ListUserOrders(context.Background(), 2)
	if err != nil {
		t.Fatalf("ListUserOrders returned error: %v", err)
	}
	if len(orders) != 1 || orders[0].Number != "12345678903" {
		t.Fatalf("orders = %+v", orders)
	}
}

func TestListPendingOrders(t *testing.T) {
	now := time.Unix(10, 0)
	storage := &Storage{pool: &fakeDB{rows: &fakeRows{rows: [][]any{
		{int64(1), "12345678903", int64(2), domain.OrderStatusProcessing, (*int64)(nil), now, now},
	}}}}

	orders, err := storage.ListPendingOrders(context.Background(), 10)
	if err != nil {
		t.Fatalf("ListPendingOrders returned error: %v", err)
	}
	if len(orders) != 1 || orders[0].Status != domain.OrderStatusProcessing {
		t.Fatalf("orders = %+v", orders)
	}
}

func TestUpdateOrderAccrual(t *testing.T) {
	points := domain.Points(5050)
	db := &fakeDB{}
	storage := &Storage{pool: db}

	err := storage.UpdateOrderAccrual(context.Background(), "12345678903", domain.OrderStatusProcessed, &points)
	if err != nil {
		t.Fatalf("UpdateOrderAccrual returned error: %v", err)
	}
	if got := db.execs[0].args[2].(*int64); *got != 5050 {
		t.Fatalf("accrual arg = %d, want 5050", *got)
	}
}

func TestGetBalance(t *testing.T) {
	storage := &Storage{pool: &fakeDB{row: fakeRow{values: []any{int64(10000), int64(2550)}}}}

	balance, err := storage.GetBalance(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetBalance returned error: %v", err)
	}
	if balance.Current != 7450 || balance.Withdrawn != 2550 {
		t.Fatalf("balance = %+v", balance)
	}
}

func TestCreateWithdrawal(t *testing.T) {
	tx := &fakeTx{row: fakeRow{values: []any{int64(10000), int64(0)}}}
	storage := &Storage{pool: &fakeDB{tx: tx}}

	err := storage.CreateWithdrawal(context.Background(), domain.Withdrawal{
		UserID:      1,
		OrderNumber: "12345678903",
		Sum:         2500,
	})
	if err != nil {
		t.Fatalf("CreateWithdrawal returned error: %v", err)
	}
	if !tx.committed {
		t.Fatal("transaction was not committed")
	}
}

func TestCreateWithdrawalInsufficientFunds(t *testing.T) {
	storage := &Storage{pool: &fakeDB{tx: &fakeTx{row: fakeRow{values: []any{int64(1000), int64(0)}}}}}

	err := storage.CreateWithdrawal(context.Background(), domain.Withdrawal{
		UserID:      1,
		OrderNumber: "12345678903",
		Sum:         2500,
	})
	if !errors.Is(err, domain.ErrInsufficientFunds) {
		t.Fatalf("CreateWithdrawal error = %v, want %v", err, domain.ErrInsufficientFunds)
	}
}

func TestListWithdrawals(t *testing.T) {
	now := time.Unix(10, 0)
	storage := &Storage{pool: &fakeDB{rows: &fakeRows{rows: [][]any{
		{int64(1), int64(2), "12345678903", int64(2500), now},
	}}}}

	withdrawals, err := storage.ListWithdrawals(context.Background(), 2)
	if err != nil {
		t.Fatalf("ListWithdrawals returned error: %v", err)
	}
	if len(withdrawals) != 1 || withdrawals[0].Sum != 2500 {
		t.Fatalf("withdrawals = %+v", withdrawals)
	}
}

func TestApplyMigrations(t *testing.T) {
	db := &fakeDB{}
	storage := &Storage{pool: db}

	if err := storage.applyMigrations(context.Background()); err != nil {
		t.Fatalf("applyMigrations returned error: %v", err)
	}
	if len(db.execs) < 3 {
		t.Fatalf("execs = %d, want at least 3 migrations", len(db.execs))
	}
	if !strings.Contains(db.execs[0].sql, "CREATE TABLE IF NOT EXISTS users") {
		t.Fatalf("first migration = %q", db.execs[0].sql)
	}
}

type fakeExec struct {
	sql  string
	args []any
}

type fakeDB struct {
	execs   []fakeExec
	execErr error
	row     fakeRow
	rows    *fakeRows
	tx      *fakeTx
}

func (db *fakeDB) Exec(_ context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	db.execs = append(db.execs, fakeExec{sql: sql, args: arguments})
	if db.execErr != nil {
		return pgconn.CommandTag{}, db.execErr
	}
	return pgconn.NewCommandTag("OK"), nil
}

func (db *fakeDB) Query(_ context.Context, _ string, _ ...any) (pgx.Rows, error) {
	if db.rows == nil {
		return &fakeRows{}, nil
	}
	return db.rows, nil
}

func (db *fakeDB) QueryRow(_ context.Context, _ string, _ ...any) pgx.Row {
	return db.row
}

func (db *fakeDB) Begin(_ context.Context) (pgx.Tx, error) {
	if db.tx == nil {
		db.tx = &fakeTx{}
	}
	return db.tx, nil
}

func (db *fakeDB) Ping(context.Context) error {
	return nil
}

func (db *fakeDB) Close() {}

type fakeTx struct {
	fakeDB
	row        fakeRow
	committed  bool
	rolledBack bool
}

func (tx *fakeTx) Begin(context.Context) (pgx.Tx, error) {
	return tx, nil
}

func (tx *fakeTx) Commit(context.Context) error {
	tx.committed = true
	return nil
}

func (tx *fakeTx) Rollback(context.Context) error {
	tx.rolledBack = true
	return pgx.ErrTxClosed
}

func (tx *fakeTx) QueryRow(_ context.Context, _ string, _ ...any) pgx.Row {
	return tx.row
}

func (tx *fakeTx) CopyFrom(context.Context, pgx.Identifier, []string, pgx.CopyFromSource) (int64, error) {
	return 0, nil
}

func (tx *fakeTx) SendBatch(context.Context, *pgx.Batch) pgx.BatchResults {
	return nil
}

func (tx *fakeTx) LargeObjects() pgx.LargeObjects {
	return pgx.LargeObjects{}
}

func (tx *fakeTx) Prepare(context.Context, string, string) (*pgconn.StatementDescription, error) {
	return nil, nil
}

func (tx *fakeTx) Conn() *pgx.Conn {
	return nil
}

type fakeRow struct {
	values []any
	err    error
}

func (r fakeRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	assignValues(dest, r.values)
	return nil
}

type fakeRows struct {
	rows   [][]any
	index  int
	err    error
	closed bool
}

func (r *fakeRows) Close() {
	r.closed = true
}

func (r *fakeRows) Err() error {
	return r.err
}

func (r *fakeRows) CommandTag() pgconn.CommandTag {
	return pgconn.NewCommandTag("SELECT 1")
}

func (r *fakeRows) FieldDescriptions() []pgconn.FieldDescription {
	return nil
}

func (r *fakeRows) Next() bool {
	if r.index >= len(r.rows) {
		r.closed = true
		return false
	}
	r.index++
	return true
}

func (r *fakeRows) Scan(dest ...any) error {
	assignValues(dest, r.rows[r.index-1])
	return nil
}

func (r *fakeRows) Values() ([]any, error) {
	return r.rows[r.index-1], nil
}

func (r *fakeRows) RawValues() [][]byte {
	return nil
}

func (r *fakeRows) Conn() *pgx.Conn {
	return nil
}

func assignValues(dest []any, values []any) {
	for i := range dest {
		switch target := dest[i].(type) {
		case *int64:
			*target = values[i].(int64)
		case **int64:
			if values[i] == nil {
				*target = nil
			} else {
				value := values[i].(*int64)
				*target = value
			}
		case *string:
			*target = values[i].(string)
		case *domain.OrderStatus:
			*target = values[i].(domain.OrderStatus)
		case *time.Time:
			*target = values[i].(time.Time)
		}
	}
}
