package gae

import (
	"context"
	"errors"
	"log"

	"cloud.google.com/go/datastore"
)

type DataStore[T any] struct {
	DSClient      *datastore.Client
	AutoCreateKey bool
	KeyGetter     func(*T) *datastore.Key
	KeySetter     func(*T, *datastore.Key)
	ToDBValue     func(*T) map[string]interface{}
	FromDBValue   func(map[string]interface{}) T
	GetByKeyQuery func(key string) datastore.Query

	kind    string
	context context.Context
}

func NewDataStore[T any](kind string, dsclient *datastore.Client) *DataStore[T] {
	return &DataStore[T]{
		DSClient: dsclient,
	}
}

func (ds *DataStore[T]) Kind() string {
	return ds.kind
}

func (ds *DataStore[T]) GetEntityKey(entity *T) string {
	return ""
}

func (ds *DataStore[T]) SetEntityKey(entity *T, key string) error {
	if ds.AutoCreateKey {
		return errors.New("KeySetter func must be set when keys can be auto created")
	}
	return nil
}

func (ds *DataStore[T]) GetByKey(key string, ctx context.Context) (*T, error) {
	ctx = ds.ensureContext(ctx)
	var out T
	dbkey := datastore.NameKey(ds.kind, key, nil)
	err := ds.DSClient.Get(ctx, dbkey, &out)
	return &out, err
}

func (ds *DataStore[T]) DeleteByKey(key string, ctx context.Context) error {
	ctx = ds.ensureContext(ctx)
	dbkey := datastore.NameKey(ds.kind, key, nil)
	log.Println("DBKey: ", dbkey)
	err := ds.DSClient.Delete(ctx, dbkey)
	return err
}

func (ds *DataStore[T]) ensureContext(ctx context.Context) context.Context {
	if ctx == nil {
		if ds.context == nil {
			ds.context = context.Background()
		}
		ctx = ds.context
	}
	return ctx
}

func (ds *DataStore[T]) SaveEntity(entity *T, ctx context.Context) (*T, *datastore.Key, error) {
	ctx = ds.ensureContext(ctx)
	newKey := datastore.IncompleteKey(ds.kind, nil)
	entityKey := ds.KeyGetter(entity)
	if entityKey == nil {
		if !ds.AutoCreateKey {
			return nil, nil, errors.New(`Key cannot be autocreated for ${ds.kind}`)
		} else {
			_, err := ds.DSClient.Put(ctx, newKey, entity)
			if err != nil {
				return nil, nil, err
			}
			ds.KeySetter(entity, newKey)
		}
	} else {
		newKey = entityKey
	}

	// Now update with the
	ds.DSClient.Put(ctx, newKey, entity)

	return entity, newKey, nil
}

func (ds *DataStore[T]) ListEntities(offset int, count int, ctx context.Context) ([]T, bool, error) {
	ctx = ds.ensureContext(ctx)
	query := datastore.NewQuery(ds.kind)
	query = query.Offset(offset)
	query = query.Limit(count + 1)
	var entities []T
	_, err := ds.DSClient.GetAll(ctx, query, entities)
	if entities == nil || err != nil {
		return nil, false, err
	}

	hasMore := len(entities) > count
	return entities, hasMore, err
}
