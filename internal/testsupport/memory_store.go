package testsupport

import (
	"context"
	"sync"
	"time"

	"github.com/glycoview/nightscout-api/model"
	"github.com/glycoview/nightscout-api/store"
)

type MemoryStore struct {
	mu      sync.RWMutex
	current map[string]map[string]model.Record
	history map[string][]model.Record
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		current: map[string]map[string]model.Record{},
		history: map[string][]model.Record{},
	}
}

func (s *MemoryStore) Create(_ context.Context, collection string, data map[string]any, subject string) (model.Record, bool, error) {
	collection = model.NormalizeCollection(collection)
	clean, err := store.NormalizeData(collection, data)
	if err != nil {
		return model.Record{}, false, err
	}
	identifier := recordIdentifier(clean)
	if identifier == "" {
		identifier = store.CalculateIdentifier(clean)
	}
	if identifier == "" {
		identifier = store.GenerateIdentifier()
	}
	clean["identifier"] = identifier
	clean["_id"] = identifier

	s.mu.Lock()
	defer s.mu.Unlock()

	if existing, ok := s.collection(collection)[identifier]; ok {
		return existing.Clone(), false, nil
	}
	dedupeKey := store.DedupeKey(collection, clean)
	if dedupeKey != "" {
		for _, existing := range s.collection(collection) {
			if !existing.IsValid {
				continue
			}
			if store.DedupeKey(collection, existing.Data) == dedupeKey {
				return existing.Clone(), false, nil
			}
		}
	}

	now := time.Now().UnixMilli()
	record := model.Record{
		Collection:  collection,
		ID:          identifier,
		Data:        clean,
		SrvCreated:  now,
		SrvModified: now,
		Subject:     subject,
		IsValid:     true,
	}
	s.collection(collection)[identifier] = record
	s.appendHistory(collection, record)
	return record.Clone(), true, nil
}

func (s *MemoryStore) Get(_ context.Context, collection, identifier string) (model.Record, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	record, ok := s.collection(model.NormalizeCollection(collection))[identifier]
	if !ok {
		return model.Record{}, store.ErrNotFound
	}
	if !record.IsValid {
		return model.Record{}, store.ErrGone
	}
	return record.Clone(), nil
}

func (s *MemoryStore) Search(_ context.Context, collection string, query store.Query) ([]model.Record, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	collection = model.NormalizeCollection(collection)
	query = withDefaultQuery(query)
	records := make([]model.Record, 0, len(s.collection(collection)))
	for _, record := range s.collection(collection) {
		records = append(records, record.Clone())
	}
	return store.ApplyQuery(records, query), nil
}

func (s *MemoryStore) Replace(_ context.Context, collection, identifier string, data map[string]any, subject string) (model.Record, bool, error) {
	collection = model.NormalizeCollection(collection)
	clean, err := store.NormalizeData(collection, data)
	if err != nil {
		return model.Record{}, false, err
	}
	clean["identifier"] = identifier
	clean["_id"] = identifier

	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UnixMilli()
	existing, ok := s.collection(collection)[identifier]
	if ok {
		if !existing.IsValid {
			return model.Record{}, false, store.ErrGone
		}
		record := existing.Clone()
		record.Data = clean
		record.SrvModified = now
		record.Subject = subject
		s.collection(collection)[identifier] = record
		s.appendHistory(collection, record)
		return record.Clone(), false, nil
	}

	record := model.Record{
		Collection:  collection,
		ID:          identifier,
		Data:        clean,
		SrvCreated:  now,
		SrvModified: now,
		Subject:     subject,
		IsValid:     true,
	}
	s.collection(collection)[identifier] = record
	s.appendHistory(collection, record)
	return record.Clone(), true, nil
}

func (s *MemoryStore) Patch(_ context.Context, collection, identifier string, patch map[string]any, subject string) (model.Record, error) {
	collection = model.NormalizeCollection(collection)
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.collection(collection)[identifier]
	if !ok {
		return model.Record{}, store.ErrNotFound
	}
	if !existing.IsValid {
		return model.Record{}, store.ErrGone
	}
	merged := model.Merge(existing.Data, patch)
	merged["identifier"] = identifier
	merged["_id"] = identifier
	merged["modifiedBy"] = subject
	clean, err := store.NormalizeData(collection, merged)
	if err != nil {
		return model.Record{}, err
	}
	clean["identifier"] = identifier
	clean["_id"] = identifier
	clean["modifiedBy"] = subject
	record := existing.Clone()
	record.Data = clean
	record.SrvModified = time.Now().UnixMilli()
	s.collection(collection)[identifier] = record
	s.appendHistory(collection, record)
	return record.Clone(), nil
}

func (s *MemoryStore) Delete(_ context.Context, collection, identifier string, permanent bool, _ string) error {
	collection = model.NormalizeCollection(collection)
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.collection(collection)[identifier]
	if !ok {
		return store.ErrNotFound
	}
	if permanent {
		record = record.Clone()
		record.IsValid = false
		now := time.Now().UnixMilli()
		record.SrvModified = now
		record.DeletedAt = &now
		s.appendHistory(collection, record)
		delete(s.collection(collection), identifier)
		return nil
	}
	if !record.IsValid {
		return store.ErrGone
	}
	now := time.Now().UnixMilli()
	record = record.Clone()
	record.IsValid = false
	record.SrvModified = now
	record.DeletedAt = &now
	s.collection(collection)[identifier] = record
	s.appendHistory(collection, record)
	return nil
}

func (s *MemoryStore) DeleteMatching(ctx context.Context, collection string, query store.Query, permanent bool, subject string) (int, error) {
	query.IncludeDeleted = true
	records, err := s.Search(ctx, collection, query)
	if err != nil {
		return 0, err
	}
	deleted := 0
	for _, record := range records {
		if !record.IsValid && !permanent {
			continue
		}
		if err := s.Delete(ctx, collection, record.Identifier(), permanent, subject); err != nil && err != store.ErrGone {
			return deleted, err
		}
		deleted++
	}
	return deleted, nil
}

func (s *MemoryStore) History(_ context.Context, collection string, since int64, limit int) ([]model.Record, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	collection = model.NormalizeCollection(collection)
	records := make([]model.Record, 0, len(s.history[collection]))
	for _, record := range s.history[collection] {
		if record.SrvModified >= since {
			records = append(records, record.Clone())
		}
	}
	// Keep the test store simple; full ordering is exercised elsewhere.
	if limit > 0 && len(records) > limit {
		records = records[len(records)-limit:]
	}
	return records, nil
}

func (s *MemoryStore) LastModified(_ context.Context, collections []string) (map[string]int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make(map[string]int64, len(collections))
	for _, name := range collections {
		collection := model.NormalizeCollection(name)
		var latest int64
		for _, record := range s.history[collection] {
			if record.SrvModified > latest {
				latest = record.SrvModified
			}
		}
		result[collection] = latest
	}
	return result, nil
}

func (s *MemoryStore) collection(name string) map[string]model.Record {
	if _, ok := s.current[name]; !ok {
		s.current[name] = map[string]model.Record{}
	}
	return s.current[name]
}

func (s *MemoryStore) appendHistory(collection string, record model.Record) {
	s.history[collection] = append(s.history[collection], record.Clone())
}

func recordIdentifier(data map[string]any) string {
	if identifier, ok := model.StringField(data, "identifier"); ok && identifier != "" {
		return identifier
	}
	if identifier, ok := model.StringField(data, "_id"); ok && identifier != "" {
		return identifier
	}
	return ""
}

func withDefaultQuery(query store.Query) store.Query {
	defaults := store.DefaultQuery()
	if query.SortField == "" {
		query.SortField = defaults.SortField
	}
	if query.Limit == 0 {
		query.Limit = defaults.Limit
	}
	return query
}
