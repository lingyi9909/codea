package update

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type TxStatus string

const (
	TxPending    TxStatus = "pending"
	TxCommitted  TxStatus = "committed"
	TxRolledBack TxStatus = "rolled_back"
)

type TxStep struct {
	Name      string    `json:"name"`
	Status    TxStatus  `json:"status"`
	StartedAt time.Time `json:"startedAt"`
	Error     string    `json:"error,omitempty"`
}

type Transaction struct {
	ID               string    `json:"id"`
	FromVersion      string    `json:"fromVersion"`
	ToVersion        string    `json:"toVersion"`
	Status           TxStatus  `json:"status"`
	StartedAt        time.Time `json:"startedAt"`
	Steps            []TxStep  `json:"steps"`
	ConfigBackupPath string    `json:"configBackupPath,omitempty"`
}

func (tx *Transaction) MarkStep(name string, status TxStatus, message string) {
	now := time.Now().UTC()
	for i := range tx.Steps {
		if tx.Steps[i].Name == name {
			tx.Steps[i].Status = status
			tx.Steps[i].Error = message
			if tx.Steps[i].StartedAt.IsZero() {
				tx.Steps[i].StartedAt = now
			}
			return
		}
	}
	tx.Steps = append(tx.Steps, TxStep{Name: name, Status: status, StartedAt: now, Error: message})
}

func (tx *Transaction) StepStatus(name string) (TxStatus, bool) {
	for _, step := range tx.Steps {
		if step.Name == name {
			return step.Status, true
		}
	}
	return "", false
}

func (tx *Transaction) StepCommitted(name string) bool {
	status, ok := tx.StepStatus(name)
	return ok && status == TxCommitted
}

type Journal struct{ path string }

func NewJournal(homeDir string) *Journal {
	return &Journal{path: filepath.Join(homeDir, "update_journal.json")}
}

func (j *Journal) Begin(tx *Transaction) error {
	if tx == nil {
		return errors.New("transaction is nil")
	}
	if tx.ID == "" || tx.ToVersion == "" {
		return errors.New("transaction id/toVersion required")
	}
	if tx.StartedAt.IsZero() {
		tx.StartedAt = time.Now().UTC()
	}
	tx.Status = TxPending
	return j.Save(tx)
}

func (j *Journal) Save(tx *Transaction) error {
	if tx == nil {
		return errors.New("transaction is nil")
	}
	if err := os.MkdirAll(filepath.Dir(j.path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(tx, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	tmp, err := os.CreateTemp(filepath.Dir(j.path), ".update-journal-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := replaceFileAtomic(tmpName, j.path); err != nil {
		return fmt.Errorf("commit journal: %w", err)
	}
	return nil
}

func (j *Journal) Load() (*Transaction, error) {
	data, err := os.ReadFile(j.path)
	if err != nil {
		return nil, err
	}
	var tx Transaction
	if err := json.Unmarshal(data, &tx); err != nil {
		return nil, fmt.Errorf("decode journal: %w", err)
	}
	return &tx, nil
}

func (j *Journal) RecoverPending() (*Transaction, error) {
	tx, err := j.Load()
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if tx.Status == TxPending {
		return tx, nil
	}
	return nil, nil
}
