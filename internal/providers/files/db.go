package main

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/abenz1267/elephant/v2/pkg/common"
	bolt "go.etcd.io/bbolt"
)

var (
	db         *bolt.DB
	bucketName = "files"
)

func openDB() error {
	path := common.CacheFile("files.db")

	os.Remove(path)

	var err error

	db, err = bolt.Open(path, 0o600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return err
	}

	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(bucketName))
		return err
	})

	return err
}

func putFileBatch(files []File) error {
	return db.Batch(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))

		for _, f := range files {
			data, err := f.MarshalMsg(nil)
			if err != nil {
				return err
			}

			if err := b.Put([]byte(f.Identifier), data); err != nil {
				return err
			}
		}

		return nil
	})
}

func putFile(f File) {
	err := db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))

		data, err := f.MarshalMsg(nil)
		if err != nil {
			return err
		}
		return b.Put([]byte(f.Identifier), data)
	})
	if err != nil {
		slog.Error(Name, "put", err)
	}
}

func getFile(identifier string) *File {
	var f File

	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))

		v := b.Get([]byte(identifier))
		if v == nil {
			return fmt.Errorf("file not found: %s", identifier)
		}

		_, err := f.UnmarshalMsg(v)
		return err
	})
	if err != nil {
		slog.Error(Name, "delete", err)
		return nil
	}

	return &f
}

func deleteFile(identifier string) error {
	return db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))
		return b.Delete([]byte(identifier))
	})
}

type Result struct {
	f         *File
	positions []int32
	start     int32
	score     int32
}

func getFilesByQuery(query string, exact bool) []Result {
	start := time.Now()

	var result []Result

	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))
		c := b.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {
			var f File
			if _, err := f.UnmarshalMsg(v); err != nil {
				continue
			}

			if query != "" {
				score, positions, s := common.FuzzyScore(query, f.Path, exact)
				if score > config.MinScore {
					result = append(result, Result{
						score:     score,
						f:         &f,
						positions: positions,
						start:     s,
					})
				}
			} else {
				if !strings.HasSuffix(f.Path, "/") {
					score := calcScore(f.Changed, start)
					result = append(result, Result{
						score: score,
						f:     &f,
					})
				}
			}
		}
		return nil
	})
	if err != nil {
		slog.Error(Name, "read", err)
		return nil
	}

	return result
}

func deleteFileByPath(path string) {
	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketName))

		return b.ForEach(func(k, v []byte) error {
			var f File
			if _, err := f.UnmarshalMsg(v); err != nil {
				return err
			}

			if strings.HasPrefix(f.Path, path) {
				deleteFile(f.Identifier)
			}

			return nil
		})
	})
	if err != nil {
		slog.Error(Name, "delete", err)
	}
}
