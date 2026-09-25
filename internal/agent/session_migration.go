package agent

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// migrate keeps history.lock until the imported session and its selection are
// durable. The marker fixes the destination across retries; it is not an index.
func (s *Sessions) migrate() (err error) {
	legacy, err := openSessionFile(filepath.Join(s.dir, "history.jsonl"), filepath.Join(s.dir, "history.lock"), s.registry, false)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, legacy.Close()) }()
	if _, err = legacy.file.Seek(0, io.SeekStart); err != nil {
		return err
	}
	data, err := io.ReadAll(io.LimitReader(legacy.file, maxHistoryBytes+1))
	if err != nil {
		return err
	}
	if len(data) > maxHistoryBytes {
		return errors.New("Legacy history exceeds the 64 MiB limit")
	}
	markerPath := filepath.Join(s.dir, "history-migration.json")
	marker, err := readSessionPointer(markerPath)
	if errors.Is(err, os.ErrNotExist) {
		marker = sessionPointer{Version: 1, ID: newHistoryID()}
		raw, _ := json.Marshal(marker)
		if err = s.writeFile(markerPath, append(raw, '\n'), false); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	target := s.path(marker.ID)
	existing, err := openPrivateFile(target, 0)
	if err == nil {
		previous, readErr := io.ReadAll(io.LimitReader(existing, maxHistoryBytes+1))
		readErr = errors.Join(readErr, existing.Close())
		if readErr != nil {
			return readErr
		}
		if !bytes.Equal(previous, data) {
			s.selectionNotice(errors.New("Migration target differs from the legacy archive; it was left unchanged"))
			return nil
		}
	} else if errors.Is(err, os.ErrNotExist) {
		if err = s.writeFile(target, data, false); err != nil {
			return err
		}
	} else {
		return err
	}
	if err = s.resume(marker.ID); err != nil {
		return err
	}
	if err = s.finishMigration(); err != nil {
		return err
	}
	s.notice = legacy.notice + s.notice
	return nil
}

func (s *Sessions) finishMigration() error {
	if err := os.Remove(filepath.Join(s.dir, "history-migration.json")); err != nil {
		return err
	}
	if err := s.syncDir(s.dir); err != nil {
		return err
	}
	s.notice = fmt.Sprintf("Imported legacy history into session %s. The archive at ~/.mino/history.jsonl is retained and is not removed by /clear. Older Mino versions do not share the new sessions.\n", s.id)
	return nil
}
