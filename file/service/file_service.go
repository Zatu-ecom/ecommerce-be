package service

import "ecommerce-be/file/repository"

type FileService interface {
}

type fileService struct {
	fileRepo repository.FileRepository
}

func NewFileService(fileRepo repository.FileRepository) FileService {
	return &fileService{
		fileRepo: fileRepo,
	}
}
