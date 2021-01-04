package storage

import "context"

type MockedStorage struct{}

func (MockedStorage) SaveSnapshot(_ context.Context, _ SaveRequest) error {
	return nil
}

func (MockedStorage) SaveRecovery(_ context.Context, _ SaveRequest) error {
	return nil
}

func (MockedStorage) GetClusterLastSnapshot(_ context.Context, _ string) ([]byte, error) {
	return nil, nil
}

func (MockedStorage) GetRecoveries(_ context.Context, _ string) ([][]byte, error) {
	return nil, nil
}
