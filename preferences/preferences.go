package preferences

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"io"
	"log"
)

type preference struct {
	value  string
	secure bool
}

// Preferences object type
type Preferences struct {
	preferences map[string]preference
	passphrase  string
	watchers    map[string][]func(*string, *string)
	dbFile      *string
}

// SetBackingStore sets the database backing store
func (p *Preferences) SetBackingStore(dbFile string) {
	validate(p)

	// Read any preferences from the DB connection
	log.Printf("Reading preferences from '%s'", dbFile)
	db, err := sql.Open("sqlite3", dbFile)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	sqlStmt := `
	create table IF NOT EXISTS preferences (name text not null primary key, value text, secure bool);
	`
	_, err = db.Exec(sqlStmt)
	if err != nil {
		log.Printf("%q: %s\n", err, sqlStmt)
		return
	}

	rows, err := db.Query("select name, value, secure from preferences")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		var value string
		var secure bool
		err = rows.Scan(&name, &value, &secure)
		if err != nil {
			log.Fatal(err)
		}
		if secure {
			rawBytes, err := base64.StdEncoding.DecodeString(value)
			if nil != err {
				log.Fatal(err)
			}
			value = string(decrypt(rawBytes, p.passphrase))
		}
		// This won't attempt to save in the DB because we haven't saved a
		// reference to the DB yet
		p.Set(name, value, secure)
	}
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}

	// Save the DB reference
	p.dbFile = &dbFile
}

// Get the value of the named preference
func (p *Preferences) Get(name string) (*string, bool) {
	validate(p)

	pref, prs := p.preferences[name]
	if prs {
		var value string

		// Check for secure values
		if pref.secure {
			rawBytes, err := base64.StdEncoding.DecodeString(pref.value)
			if nil != err {
				log.Fatal(err)
			}
			decrypted := decrypt(rawBytes, p.passphrase)
			value = string(decrypted)
		} else {
			// Otherwise copy the string because we're returning a pointer
			value = string(pref.value)
		}

		return &value, true
	}
	return nil, false
}

// Set the value of the named preference
func (p *Preferences) Set(name string, value string, secure ...bool) {
	validate(p)

	pref := preference{}
	if len(secure) > 0 && secure[0] {
		pref.secure = true
		rawBytes := encrypt([]byte(value), p.passphrase)
		pref.value = base64.StdEncoding.EncodeToString(rawBytes)
	} else {
		pref.value = value
	}

	_value, _ := p.Get(name)
	p.preferences[name] = pref

	// Save the value to the DB
	if nil != p.dbFile {
		db, err := sql.Open("sqlite3", *p.dbFile)
		if err != nil {
			log.Fatal(err)
		}
		defer db.Close()
		stmt, err := db.Prepare("insert into preferences(name, value, secure) values(?, ?, ?) on conflict(name) do update set value = ?, secure = ?;")
		if err != nil {
			log.Fatal(err)
		}
		defer stmt.Close()
		_, err = stmt.Exec(name, pref.value, pref.secure, pref.value, pref.secure)
		if err != nil {
			log.Fatal(err)
		}
	}

	// Notify all watchers
	watchers, exists := p.watchers[name]
	if exists {
		for _, f := range watchers {
			f(_value, &value)
		}
	}
}

// SetDefaultPreference will set the preference value if no value is currently set
func (p *Preferences) SetDefaultPreference(name string, value string, secure ...bool) {
	validate(p)

	_, prs := p.Get(name)
	if !prs {
		_secure := false
		if len(secure) > 0 {
			_secure = secure[0]
		}
		p.Set(name, value, _secure)
	}
}

// AddWatcher adds a watcher function for the given name
func (p *Preferences) AddWatcher(name string, watcher func(*string, *string)) {
	validate(p)

	watchers, exists := p.watchers[name]
	if !exists {
		watchers = make([]func(*string, *string), 0)
		p.watchers[name] = watchers
	}
	p.watchers[name] = append(watchers, watcher)
}

func validate(p *Preferences) {
	if nil == p.preferences {
		p.preferences = make(map[string]preference)
	}

	if nil == p.watchers {
		p.watchers = make(map[string][]func(*string, *string))
	}

	if 0 == len(p.passphrase) {
		p.passphrase = "PleaseChangeMe"
	}
}

// https://www.thepolyglotdeveloper.com/2018/02/encrypt-decrypt-data-golang-application-crypto-packages/
func createHash(key string) string {
	hasher := md5.New()
	hasher.Write([]byte(key))
	return hex.EncodeToString(hasher.Sum(nil))
}

func encrypt(data []byte, passphrase string) []byte {
	block, _ := aes.NewCipher([]byte(createHash(passphrase)))
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		panic(err.Error())
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		panic(err.Error())
	}
	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return ciphertext
}

func decrypt(data []byte, passphrase string) []byte {
	key := []byte(createHash(passphrase))
	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err.Error())
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		panic(err.Error())
	}
	nonceSize := gcm.NonceSize()
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		panic(err.Error())
	}
	return plaintext
}
