package main

import (
	"archive/zip"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
)

type CitiesLookup struct {
	ISO   string
	Citys []City
}

func New(iso string) (CitiesLookup, error) {
	al := CitiesLookup{ISO: iso}
	LoadFromGeonamesOrg("cities1000.zip", "readme.txt")
	err := al.LoadCitys("./cities1000.zip", "cities1000.txt")
	if err != nil {
		logger.Fatal("Failed to load data:", err)
		return al, err
	}
	return al, nil
}

func (l *CitiesLookup) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(l.Citys)
}

var logger *log.Logger
var errorlog *os.File

func main() {
	var port = flag.Int("p", 8080, "Service port.")
	var iso = flag.String("iso", "", "Set iso code only load that iso, else all will be loaded.")
	flag.Parse() // parse the flags

	errorlog, err := os.OpenFile("citieslookup.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		logger.Printf("error opening file: %v", err)
		os.Exit(1)
	}
	defer errorlog.Close()
	logger = log.New(errorlog, "applog: ", log.Lshortfile|log.LstdFlags)

	if service, err := New(*iso); err == nil {
		mux := http.NewServeMux()

		mux.Handle("/cities", &service)
		log.Printf("Server exposes url: http://localhost:%v/cities\n", *port)
		log.Printf("Server started successfully and listen on port: %v \n", *port)
		logger.Printf("Server started successfully and listen on port: %v \n", *port)
		logger.Fatal(http.ListenAndServe(":"+strconv.Itoa(*port), mux))
	}
}

func LoadFromGeonamesOrg(res ...string) error {
	get := func(resource string, url string) error {
		response, err := http.Get(url + resource)
		if err != nil {
			return err
		}
		defer response.Body.Close()

		file, err := os.OpenFile(resource, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		defer file.Close()
		n, err := io.Copy(file, response.Body)

		//n, err := io.Copy(file, resp.Body) // first var shows number of bytes
		logger.Printf("Downloaded %v bytes.", n)
		if err != nil {
			return err
		}
		return nil
	}

	for _, r := range res {
		if _, err := os.Stat(r); err != nil {
			url := "http://download.geonames.org/export/dump/"
			if err := get(r, url); err != nil {
				return fmt.Errorf("Unable to fetch or store %s from %v. Failed with error: %s.\n", r, url, err)
			}
		} else {
			logger.Printf("%s already exist and is not fetched.\n", r)
		}
	}
	return nil
}

//City the serice LookupCity exposes this item in its web service.
type City struct {
	ID        int     `json:"id"`
	Name      string  `json:"-"`
	Longitude float64 `json:"longitude"`
	Latitude  float64 `json:"latitude"`
	ISO       string  `json:"-"`
}

func (al *CitiesLookup) LoadCitys(src, target string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer func() {
		if err := r.Close(); err != nil {
			panic(err)
		}
	}()
	// Closure to address file descriptors issue with all the deferred .Close() methods
	extractAndWriteFile := func(f *zip.File) ([]City, error) {
		logger.Println("Processing ", f.Name)
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		defer func() {
			if err := rc.Close(); err != nil {
				panic(err)
			}
		}()
		r := csv.NewReader(rc)
		r.Comma = '\t'
		var oneRecord City
		var allRecords []City
		var failedToReadRow []int
		index := 0
		for {
			record, err := r.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				//log.Println("Error on row:"+strconv.Itoa(index)+":", err)
				failedToReadRow = append(failedToReadRow, index)
				index++
				continue
			}
			oneRecord.ID, _ = strconv.Atoi(record[0])
			oneRecord.Name = record[1]

			oneRecord.Latitude, _ = strconv.ParseFloat(record[4], 64)
			oneRecord.Longitude, _ = strconv.ParseFloat(record[5], 64)
			oneRecord.ISO = record[8]
			allRecords = append(allRecords, oneRecord)
			index++
		}
		logger.Println("All Citys size:", len(allRecords))
		logger.Printf("Failed to read %v rows (look them up in cities1000.zip):%v\n ", len(failedToReadRow), failedToReadRow)
		if len(al.ISO) == 0 {
			return allRecords, nil
		}
		var isoRecords []City
		for _, rec := range allRecords {
			if rec.ISO == al.ISO {
				isoRecords = append(isoRecords, rec)
			}
		}
		logger.Printf("%s filtered size:%v \n", al.ISO, len(isoRecords))
		return isoRecords, nil
	}
	for _, f := range r.File {
		if f.FileHeader.Name == target {
			Citys, err := extractAndWriteFile(f)
			if err != nil {
				return err
			}
			al.Citys = Citys
			break
		}
	}

	return nil
}
