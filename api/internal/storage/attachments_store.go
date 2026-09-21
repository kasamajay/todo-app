package storage

import (
	"os"
	"path/filepath"

	"todo-app/internal/models"
)

type AttachmentStore struct {
	*Store[models.Attachment]
	dataDir string
}

func NewAttachmentStore(path, dataDir string) (*AttachmentStore, error) {
	s, err := NewStore[models.Attachment](path)
	if err != nil {
		return nil, err
	}
	return &AttachmentStore{s, dataDir}, nil
}

func (s *AttachmentStore) ListByTask(taskID string) []models.Attachment {
	return s.Find(func(a models.Attachment) bool { return a.TaskID == taskID })
}

// BlobPath returns the on-disk path for an attachment's binary content.
func (s *AttachmentStore) BlobPath(attachmentID string) string {
	return filepath.Join(s.dataDir, "attachments", attachmentID)
}

func (s *AttachmentStore) WriteBlob(attachmentID string, content []byte) error {
	return WriteFileAtomic(s.BlobPath(attachmentID), content)
}

func (s *AttachmentStore) ReadBlob(attachmentID string) ([]byte, error) {
	return os.ReadFile(s.BlobPath(attachmentID))
}

func (s *AttachmentStore) DeleteBlob(attachmentID string) error {
	err := os.Remove(s.BlobPath(attachmentID))
	if err != nil && os.IsNotExist(err) {
		return nil
	}
	return err
}
