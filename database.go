package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"go.etcd.io/bbolt"
	"log"
	"os"
)

const dataFolderPath = "data"

type hDataBase struct {
	db *bbolt.DB
}

// itob returns an 8-byte big endian representation of v.
func itob(v int64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}

func createBucketIfNotExists(bucketName []byte, db *bbolt.DB) error {
	return db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(bucketName)
		if err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}
		return nil
	})
}

func (db *hDataBase) initDB(wipeBucket *string) {
	_, err := os.Stat(dataFolderPath)
	if err != nil {
		if os.IsNotExist(err) {
			os.Mkdir(dataFolderPath, 0755)
		} else {
			log.Fatal(err)
		}
	}
	db.db, err = bbolt.Open("data/horae.db", 0600, nil)
	if err != nil {
		log.Fatal(err)
	}

	var requiredBuckets = []string{"users"}
	for _, bucketName := range requiredBuckets {
		err := createBucketIfNotExists([]byte(bucketName), db.db)
		if err != nil {
			log.Fatal(err)
		}
	}

	if *wipeBucket != "" {
		err = db.wipeBucket([]byte(*wipeBucket))
		if err != nil {
			log.Fatal(err)
		}
	}
}

func (db *hDataBase) saveUserData(chatId ChatId, user User) error {
	return db.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("users"))
		jsonBuf, err := json.Marshal(user)
		if err != nil {
			return fmt.Errorf("marshal user: %s", err)
		}
		err = b.Put(itob(int64(chatId)), jsonBuf)
		if err != nil {
			return fmt.Errorf("save user data: %s", err)
		}
		return nil
	})
}

func (db *hDataBase) getAllUsersData() (map[ChatId]User, error) {
	users := make(map[ChatId]User, 0)
	err := db.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("users"))
		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var user User
			err := json.Unmarshal(v, &user)
			if err != nil {
				return fmt.Errorf("unmarshal user: %s", err)
			}
			chatId := ChatId(binary.BigEndian.Uint64(k))
			users[chatId] = user
		}
		return nil
	})
	if err != nil {
		return users, err
	}
	return users, nil
}

func (db *hDataBase) wipeBucket(bucketName []byte) error {
	err := db.db.Update(func(tx *bbolt.Tx) error {
		err := tx.DeleteBucket(bucketName)
		if err != nil {
			return fmt.Errorf("delete bucket: %s", err)
		}
		fmt.Println("Bucket deleted")
		return nil
	})
	if err != nil {
		log.Fatalf("Error: %s", err)
		return err
	}
	return db.db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucket(bucketName)
		if err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}
		fmt.Println("Bucket created")
		return nil
	})
}

func (db *hDataBase) closeDB() {
	db.db.Close()
}