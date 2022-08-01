package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

type Download struct {
	Url           string
	targetPath    string
	TotalSections int
}

func (d Download) Do() error {
	fmt.Println("Making connection")
	r, err := d.getNewRequest("HEAD")
	if err != nil {
		return err
	}
	// getting Head
	resp, err := http.DefaultClient.Do(r)
	if err != nil {
		return err
	}
	fmt.Printf("Got %v\n", resp.StatusCode)

	if resp.StatusCode > 299 {
		return fmt.Errorf("can't process, responce is %v", resp.StatusCode)
	}
	// check content length
	size, err := strconv.Atoi(resp.Header.Get("Content-Length"))
	if err != nil {
		return err
	}

	fmt.Printf("Size is %v bytes\n", size)
	// make sections
	var sections = make([][2]int, d.TotalSections)
	sectionSize := size / d.TotalSections

	fmt.Printf("Section size is %v bytes \n", sectionSize)
	fmt.Println(sections)
	
	for i := range sections {
		if i == 0 {
			// starting byte of fitst section
			sections[i][0] = 0
		} else {
			// starting byte of other sections
			sections[i][0] = sections[i-1][1] + 1
		}

		if i < d.TotalSections-1 {
			// ending byte of other sections
			sections[i][1] = sections[i][0] + sectionSize
		} else {
			// ending byte of the last section
			sections[i][1] = size - 1
		}
	}

	fmt.Println(sections)

	// download sections
	var wg sync.WaitGroup
	for i, s := range sections {
		wg.Add(1)
		i:= i
		s:= s
		go func() {
			defer wg.Done()
			err = d.downloadSection(i, s)
			if err != nil {
				panic(err)
			}
		}()
	}
	wg.Wait()
	// merge downloaded sections into one file
	err = d.mergeFiles(sections)
	if err != nil {
		return err
	}
	return nil
}

func (d Download) getNewRequest(method string) (*http.Request, error) {
	r, err := http.NewRequest(
		method,
		d.Url,
		nil,
	)
	if err != nil {
		return nil, err
	}

	r.Header.Set("User-Agent", "Silly Download Manager v001")
	return r, nil
}

func (d Download) downloadSection(i int, s [2]int) error {
	r, err := d.getNewRequest("GET")
	if err != nil {
		return err
	}
	// specific section range
	r.Header.Set("Range", fmt.Sprintf("bytes=%v-%v", s[0], s[1]))

	resp, err := http.DefaultClient.Do(r)
	if err != nil {
		return err
	}

	fmt.Printf("Downloaded %v bytes for section %v: %v\n", resp.Header.Get("Content-Length"), i, s)

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(fmt.Sprintf("section-%v.tmp", i), b, os.ModePerm)
	if err != nil {
		return err
	}

	return nil
}

func (d Download) mergeFiles(sections [][2]int) error {
	f, err := os.OpenFile(d.targetPath, os.O_CREATE | os.O_WRONLY | os.O_APPEND, os.ModePerm)
	if err != nil {
		return err
	}
	defer f.Close()
	// merging sections in a right order
	for i := range sections {
		b, err := ioutil.ReadFile(fmt.Sprintf("section-%v.tmp", i))
		if err != nil {
			return err
		}

		n, err := f.Write(b)
		if err != nil {
			return err
		}
		fmt.Printf("%v bytes merged\n", n)
	}

	return nil
}

func main() {
	startTime := time.Now()
	d := Download{
		Url:           "https://www.dropbox.com/s/gjlkrubmpf30bpf/short.mp4?dl=1",
		targetPath:    "final.mp4",
		TotalSections: 10,
	}

	err := d.Do()
	if err != nil {
		log.Fatalf("An error occured while downloading the file: %s\n", err)
	}
	fmt.Printf("Download completed in %v seconds\n", time.Since(startTime).Seconds())
}
