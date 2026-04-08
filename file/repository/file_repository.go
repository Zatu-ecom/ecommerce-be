package repository

type FileRepository interface {
}

type fileRepository struct {
}

func NewFileRepository() FileRepository {
	return &fileRepository{}
}
