package crud

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/globalsign/mgo"
)

// MongoCollectionPrefix is applied to every collection.
const MongoCollectionPrefix = "cnab_"

var _ Store = &mongoDBStore{}

type mongoDBStore struct {
	url         string
	session     *mgo.Session
	collections map[string]*mgo.Collection
	dbName      string
}

type doc struct {
	Name string `json:"name"`
	Data []byte `json:"data"`
}

// NewMongoDBStore creates a new storage engine that uses MongoDB
//
// The URL provided must point to a MongoDB server and database.
func NewMongoDBStore(url string) Store {
	db := &mongoDBStore{
		url:         url,
		collections: map[string]*mgo.Collection{},
	}
	return NewBackingStore(db)
}

func (s *mongoDBStore) Connect() error {
	dbn, err := parseDBName(s.url)
	if err != nil {
		return err
	}
	s.dbName = dbn

	session, err := mgo.Dial(s.url)
	if err != nil {
		return err
	}
	s.session = session
	return nil
}

func (s *mongoDBStore) Close() error {
	if s.session != nil {
		s.session.Close()
		s.session = nil
	}
	return nil
}

func (s *mongoDBStore) getCollection(itemType string) *mgo.Collection {
	c := s.collections[itemType]
	if c == nil {
		c = s.session.DB(s.dbName).C(MongoCollectionPrefix + itemType)
		s.collections[itemType] = c
	}
	return c
}

func (s *mongoDBStore) List(itemType string) ([]string, error) {
	collection := s.getCollection(itemType)

	var res []doc
	if err := collection.Find(nil).All(&res); err != nil {
		return []string{}, wrapErr(err)
	}
	buf := []string{}
	for _, v := range res {
		buf = append(buf, v.Name)
	}
	return buf, nil
}

func (s *mongoDBStore) Save(itemType string, name string, data []byte) error {
	collection := s.getCollection(itemType)

	return wrapErr(collection.Insert(doc{name, data}))
}

func (s *mongoDBStore) Read(itemType string, name string) ([]byte, error) {
	collection := s.getCollection(itemType)

	res := doc{}
	if err := collection.Find(map[string]string{"name": name}).One(&res); err != nil {
		if err == mgo.ErrNotFound {
			return nil, ErrRecordDoesNotExist
		}
		return []byte{}, wrapErr(err)
	}
	return res.Data, nil
}

func (s *mongoDBStore) Delete(itemType string, name string) error {
	collection := s.getCollection(itemType)

	return wrapErr(collection.Remove(map[string]string{"name": name}))
}

func wrapErr(err error) error {
	if err == nil {
		return err
	}
	return fmt.Errorf("mongo storage error: %s", err)
}

func parseDBName(dialStr string) (string, error) {
	u, err := url.Parse(dialStr)
	if err != nil {
		return "", err
	}
	if u.Path != "" {
		return strings.TrimPrefix(u.Path, "/"), nil
	}
	// If this returns empty, then the driver is supposed to substitute in the
	// default database.
	return "", nil
}
