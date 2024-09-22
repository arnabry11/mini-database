package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/jcelliott/lumber"
)


const Version = "1.0.0"

type (
	Logger interface {
		Fatal(string, ...interface{}) // variadic function
		Error(string, ...interface{})
		Warn(string, ...interface{})
		Info(string, ...interface{})
		Debug(string, ...interface{})
		Trace(string, ...interface{})
	}

	Driver struct {
		mutex sync.Mutex
		mutexes map[string]*sync.Mutex
		dir string
		log Logger
	}
)

type Options struct {
	Logger
}

func New(dir string, options *Options)(*Driver, error) {
  dir =  filepath.Clean(dir)

	opts := Options{}

	if options != nil {
		opts = *options
	}

	if opts.Logger == nil {
		opts.Logger = lumber.NewConsoleLogger(lumber.INFO)
	}

	driver := Driver{
		dir: dir,
		mutexes: make(map[string]*sync.Mutex),
		log: opts.Logger,
	}

	if _, err := os.Stat(dir); err == nil {
		opts.Logger.Debug("Using '%s' (database already exists) \n", dir)
		return &driver, nil
	}

	opts.Logger.Debug("Creating '%s' (database does not exist) \n", dir)
	return &driver, os.MkdirAll(dir, 0755)
}

func (d *Driver) Write(collection string, resource string, v interface{}) error {
  if collection == "" {
		return fmt.Errorf("Missing collection - no place to save record!")
	}

	if resource == "" {
		return fmt.Errorf("Missing resource - unable to save record (no name)!")
	}

	mutex := d.getOrCreateMutex(collection)
	mutex.Lock()
	defer mutex.Unlock() // unlock mutex after function returns

	dir := filepath.Join(d.dir, collection)
	fnlPath := filepath.Join(dir, resource + ".json")
	tmpPath := fnlPath + ".tmp"

	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	b, err := json.MarshalIndent(v, "", "\t")
	if err != nil {
		return err
	}

	b = append(b, byte('\n'))

	if err := os.WriteFile(tmpPath, b, 0644); err != nil {
		return err
	}

	return os.Rename(tmpPath, fnlPath)
}

func (d *Driver) Read(collection, resource string, v interface{}) error {
  if collection == "" {
		return fmt.Errorf("Missing collection - no place to read record!")
	}

	if resource == "" {
		return fmt.Errorf("Missing resource - unable to read record (no name)!")
	}

	record := filepath.Join(d.dir, collection, resource + ".json")

	if _, err := stat(record); err != nil {
		return err
	}

	b, err := os.ReadFile(record)
	if err != nil {
		return err
	}

	return json.Unmarshal(b, &v)
}

func (d *Driver) ReadAll(collection string)([]string, error) {
  if collection == "" {
		return nil, fmt.Errorf("Missing collection - no place to read records!")
	}

	dir := filepath.Join(d.dir, collection)

	if _, err := stat(dir); err != nil {
		return nil, err
	}

	files, _ := os.ReadDir(dir)

	var records []string

	for _, file := range files {
		b, err := os.ReadFile(filepath.Join(dir, file.Name()))

		if err != nil {
			return nil, err
		}

		records = append(records, string(b))
	}

	return records, nil
}

func (d *Driver) Delete(collection, resource string) error {
	path := filepath.Join(d.dir, collection, resource)
	mutex := d.getOrCreateMutex(collection)
	mutex.Lock()
	defer mutex.Unlock()

	dir := filepath.Join(d.dir, path)

	switch fi, err := stat(dir); {
		case fi == nil, err != nil:
			return fmt.Errorf("unable to find file or directory named: %s", path)
		case fi.Mode().IsDir():
			return os.RemoveAll(dir)
		case fi.Mode().IsRegular():
			return os.RemoveAll(dir + ".json")
	}

	return nil
}

func (d *Driver) getOrCreateMutex(collection string) *sync.Mutex {
	d.mutex.Lock()
	defer d.mutex.Unlock()
  m, ok := d.mutexes[collection]

	if !ok {
		m = &sync.Mutex{}
		d.mutexes[collection] = m
	}

	return m
}

func stat(path string)(fi os.FileInfo, err error) {
	if fi, err = os.Stat(path); os.IsNotExist(err) {
		fi, err = os.Stat(path + ".json")
	}

	return fi, err
}

type Address struct {
	City string
	State string
	Country string
	Pincode json.Number
}

type User struct {
	Name string
	Age json.Number
	Contact string
	Company string
	Address Address
}

func main() {
  dir := "./"

	db, err := New(dir, nil)

	if err != nil {
		// panic(err)
		fmt.Println("Error:", err)
	}

	employees := []User {
		{ "John", "23", "2378367837", "Google", Address{"Dhanbad", "Jharkhand", "India", "828122"} },
		{ "Doe", "25", "2378367837", "Facebook", Address{"Ranchi", "Jharkhand", "India", "828133"} },
		{ "Jane", "27", "2378367837", "Amazon", Address{"Jamshedpur", "Jharkhand", "India", "821645"} },
		{ "Dane", "29", "2378367837", "Microsoft", Address{"Jamtara", "Jharkhand", "India", "287334"} },
		{ "Pete", "31", "2378367837", "Apple", Address{"Bokaro", "Jharkhand", "India", "179232"} },
		{ "Steve", "33", "2378367837", "Tesla", Address{"Bhuli", "Jharkhand", "India", "987632"} },
	}

	for _, employee := range employees {
		db.Write("users", employee.Name, User{
			Name: employee.Name,
			Age: employee.Age,
			Contact: employee.Contact,
			Company: employee.Company,
			Address: employee.Address,
		})
	}

	records, err := db.ReadAll("users")

	if err != nil {
		fmt.Println("Error:", err)
	}

	fmt.Print(records)

	allUsers := []User{}

  for _, f := range records {
		employeeFound := User{}

		if err := json.Unmarshal([]byte(f), &employeeFound); err != nil {
			fmt.Println("Error:", err)
		}

		allUsers = append(allUsers, employeeFound)
	}

	fmt.Println(allUsers)

	// if err := db.Delete("users", "John"); err != nil {
	// 	fmt.Println("Error:", err)
	// }

	// if err := db.Delete("users", ""); err != nil {
	// 	fmt.Println("Error:", err)
	// }
}
