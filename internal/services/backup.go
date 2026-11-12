package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/marcuth/social-media-copilot/internal/db"
)

const settingLastBackup = "backup.last_at"

// FileDialogs is implemented by main with the native dialogs; it returns ""
// when the user cancels.
type FileDialogs interface {
	SaveFile(title, defaultName string) (string, error)
	OpenFile(title string) (string, error)
	Reveal(path string) error
}

type BackupInfo struct {
	Path         string `json:"path"`
	SizeBytes    int64  `json:"sizeBytes"`
	LastBackupAt string `json:"lastBackupAt"`
}

type BackupResult struct {
	Path      string `json:"path"`
	Cancelled bool   `json:"cancelled"`
}

type RestoreResult struct {
	Cancelled bool `json:"cancelled"`
	// SafetyCopy is the copy of the database made right before restoring.
	SafetyCopy string `json:"safetyCopy"`
	Profiles   int    `json:"profiles"`
	Posts      int    `json:"posts"`
}

type BackupService struct {
	DB      *sql.DB
	Path    string
	Dialogs FileDialogs
}

func (s *BackupService) GetBackupInfo() (BackupInfo, error) {
	info := BackupInfo{Path: s.Path}
	if st, err := os.Stat(s.Path); err == nil {
		info.SizeBytes = st.Size()
	}
	last, err := getSetting(s.DB, settingLastBackup, "")
	info.LastBackupAt = last
	return info, err
}

func (s *BackupService) ExportBackup() (BackupResult, error) {
	if s.Dialogs == nil {
		return BackupResult{}, errors.New("diálogo de arquivos indisponível")
	}
	name := "social-media-copilot-" + time.Now().Format("2006-01-02") + ".db"
	path, err := s.Dialogs.SaveFile("Salvar backup", name)
	if err != nil {
		return BackupResult{}, err
	}
	if path == "" {
		return BackupResult{Cancelled: true}, nil
	}
	if filepath.Ext(path) == "" {
		path += ".db"
	}
	if err := s.ExportTo(path); err != nil {
		return BackupResult{}, err
	}
	return BackupResult{Path: path}, nil
}

// ExportTo writes a consistent copy of the database, replacing path.
func (s *BackupService) ExportTo(path string) error {
	if same, _ := samePath(path, s.Path); same {
		return errors.New("escolha outro arquivo: este é o banco em uso")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("não foi possível substituir o arquivo: %w", err)
	}
	if _, err := s.DB.Exec(`VACUUM INTO ?`, path); err != nil {
		return fmt.Errorf("falha ao gerar o backup: %w", err)
	}
	return setSetting(s.DB, settingLastBackup, nowStamp())
}

func (s *BackupService) ImportBackup() (RestoreResult, error) {
	if s.Dialogs == nil {
		return RestoreResult{}, errors.New("diálogo de arquivos indisponível")
	}
	path, err := s.Dialogs.OpenFile("Escolha um backup do Social Media Copilot")
	if err != nil {
		return RestoreResult{}, err
	}
	if path == "" {
		return RestoreResult{Cancelled: true}, nil
	}
	return s.RestoreFrom(path)
}

func (s *BackupService) OpenDataFolder() error {
	if s.Dialogs == nil {
		return errors.New("indisponível")
	}
	return s.Dialogs.Reveal(filepath.Dir(s.Path))
}

// RestoreFrom replaces every table with the backup's contents. Older backups
// are migrated on a temporary copy first; a copy of the current database is
// saved next to it before anything changes.
func (s *BackupService) RestoreFrom(path string) (RestoreResult, error) {
	var res RestoreResult
	if same, _ := samePath(path, s.Path); same {
		return res, errors.New("esse é o banco em uso; escolha um arquivo de backup")
	}
	version, err := db.ReadVersion(path)
	if err != nil {
		return res, fmt.Errorf("arquivo inválido: %w", err)
	}
	if version > db.LatestVersion() {
		return res, errors.New("esse backup foi feito por uma versão mais nova do app; atualize o app antes de restaurar")
	}

	tmpDir, err := os.MkdirTemp("", "smc-restore-")
	if err != nil {
		return res, err
	}
	defer os.RemoveAll(tmpDir)
	tmp := filepath.Join(tmpDir, "backup.db")
	if err := copyFile(path, tmp); err != nil {
		return res, err
	}
	migrated, err := db.Open(tmp)
	if err != nil {
		return res, fmt.Errorf("não foi possível atualizar o backup: %w", err)
	}
	// Fold the WAL into the file so ATTACH sees every row.
	_, err = migrated.Exec(`PRAGMA wal_checkpoint(TRUNCATE)`)
	migrated.Close()
	if err != nil {
		return res, err
	}

	res.SafetyCopy = filepath.Join(filepath.Dir(s.Path), "backups",
		"antes-de-restaurar-"+time.Now().Format("2006-01-02-150405")+".db")
	if err := s.ExportTo(res.SafetyCopy); err != nil {
		return res, fmt.Errorf("não foi possível salvar a cópia de segurança: %w", err)
	}

	if err := s.copyTables(tmp); err != nil {
		return res, err
	}
	_ = s.DB.QueryRow(`SELECT COUNT(*) FROM profiles`).Scan(&res.Profiles)
	_ = s.DB.QueryRow(`SELECT COUNT(*) FROM posts`).Scan(&res.Posts)
	return res, nil
}

func (s *BackupService) copyTables(backupPath string) (err error) {
	ctx := context.Background()
	// A dedicated connection keeps ATTACH and the pragma on the same session.
	conn, err := s.DB.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, `PRAGMA foreign_keys = OFF`); err != nil {
		return err
	}
	defer conn.ExecContext(ctx, `PRAGMA foreign_keys = ON`)
	if _, err := conn.ExecContext(ctx, `ATTACH DATABASE ? AS bk`, filepath.ToSlash(backupPath)); err != nil {
		return err
	}
	defer conn.ExecContext(ctx, `DETACH DATABASE bk`)

	tables, err := tableNames(ctx, conn, "main")
	if err != nil {
		return err
	}
	backupTables, err := tableNames(ctx, conn, "bk")
	if err != nil {
		return err
	}
	inBackup := map[string]bool{}
	for _, t := range backupTables {
		inBackup[t] = true
	}

	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()
	for _, t := range tables {
		if _, err = tx.ExecContext(ctx, `DELETE FROM main."`+t+`"`); err != nil {
			return err
		}
		if !inBackup[t] {
			continue
		}
		cols, err2 := commonColumns(ctx, tx, t)
		if err2 != nil {
			return err2
		}
		list := `"` + strings.Join(cols, `", "`) + `"`
		if _, err = tx.ExecContext(ctx, `INSERT INTO main."`+t+`" (`+list+`) SELECT `+list+` FROM bk."`+t+`"`); err != nil {
			return fmt.Errorf("restaurar %s: %w", t, err)
		}
	}
	// AUTOINCREMENT counters live in sqlite_sequence.
	if _, err = tx.ExecContext(ctx, `DELETE FROM main.sqlite_sequence`); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO main.sqlite_sequence (name, seq) SELECT name, seq FROM bk.sqlite_sequence`); err != nil {
		return err
	}
	return tx.Commit()
}

type queryContexter interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

func tableNames(ctx context.Context, q queryContexter, schema string) ([]string, error) {
	rows, err := q.QueryContext(ctx, `SELECT name FROM `+schema+`.sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func columns(ctx context.Context, q queryContexter, schema, table string) (map[string]bool, []string, error) {
	rows, err := q.QueryContext(ctx, `SELECT name FROM pragma_table_info(?, ?)`, table, schema)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	set := map[string]bool{}
	var order []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			return nil, nil, err
		}
		set[n] = true
		order = append(order, n)
	}
	return set, order, rows.Err()
}

func commonColumns(ctx context.Context, q queryContexter, table string) ([]string, error) {
	_, mainCols, err := columns(ctx, q, "main", table)
	if err != nil {
		return nil, err
	}
	bkCols, _, err := columns(ctx, q, "bk", table)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, c := range mainCols {
		if bkCols[c] {
			out = append(out, c)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("tabela %s sem colunas em comum", table)
	}
	return out, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

func samePath(a, b string) (bool, error) {
	ia, err := os.Stat(a)
	if err != nil {
		return false, err
	}
	ib, err := os.Stat(b)
	if err != nil {
		return false, err
	}
	return os.SameFile(ia, ib), nil
}
