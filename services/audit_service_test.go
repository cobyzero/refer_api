package services

import (
	"testing"

	"refer_api/models"
	"refer_api/repository"
)

type stubAuditRepo struct {
	created *models.AuditLog
	list    []models.AuditLog
	total   int64
	actions []string
}

func (s *stubAuditRepo) Create(e *models.AuditLog) error {
	s.created = e
	return nil
}
func (s *stubAuditRepo) List(_ repository.AuditFilter) ([]models.AuditLog, int64, error) {
	return s.list, s.total, nil
}
func (s *stubAuditRepo) DistinctActions(_ int) ([]string, error) {
	return s.actions, nil
}

func TestAuditService_LogShapesEntry(t *testing.T) {
	stub := &stubAuditRepo{}
	svc := NewAuditService(stub)
	err := svc.Log("u1", "a@a.com", []string{"ADMIN", "USER"}, "order.create", "order", "o1", 201, "127.0.0.1", map[string]interface{}{"total": 9.5})
	if err != nil {
		t.Fatalf("Log: %v", err)
	}
	e := stub.created
	if e.Action != "order.create" || e.Status != 201 {
		t.Errorf("bad action/status: %+v", e)
	}
	if e.UserID == nil || *e.UserID != "u1" {
		t.Errorf("bad userId: %+v", e.UserID)
	}
	if e.Roles == nil || *e.Roles != "ADMIN,USER" {
		t.Errorf("bad roles: %+v", e.Roles)
	}
	if e.Meta["total"] != 9.5 {
		t.Errorf("bad meta: %v", e.Meta)
	}
	// Empty strings must not produce empty-string pointers.
	if err := svc.Log("", "", nil, "x", "", "", 0, "", nil); err != nil {
		t.Fatalf("Log empty: %v", err)
	}
	if stub.created.UserID != nil || stub.created.Email != nil || stub.created.Roles != nil {
		t.Errorf("expected nil optionals, got %+v", stub.created)
	}
}

func TestAuditService_ListPassthrough(t *testing.T) {
	stub := &stubAuditRepo{list: []models.AuditLog{{Action: "a"}}, total: 1, actions: []string{"a", "b"}}
	svc := NewAuditService(stub)
	data, total, err := svc.List(repository.AuditFilter{Page: 1, Limit: 20})
	if err != nil || total != 1 || len(data) != 1 {
		t.Fatalf("List: %v %d %d", err, total, len(data))
	}
	actions, err := svc.Actions()
	if err != nil || len(actions) != 2 {
		t.Fatalf("Actions: %v %v", err, actions)
	}
}
