package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

type File struct {
	url          string
	pathToSave   string
	noOfSections int
}

func main() {
	start := time.Now()
	d := File{
		url:          "https://r4---sn-35153iuxa-3b4e.googlevideo.com/videoplayback?expire=1635285535&ei=vyV4YYDIOI3h7gPjv5q4Bw&ip=178.89.181.198&id=o-AOcM-KfnhXHq93jkwuIsdD_XJcAw7m3Xt4tRjbfSx2OB&itag=22&source=youtube&requiressl=yes&mh=y4&mm=31%2C29&mn=sn-35153iuxa-3b4e%2Csn-35153iuxa-unxd&ms=au%2Crdu&mv=m&mvi=4&pl=22&initcwndbps=527500&vprv=1&mime=video%2Fmp4&ns=JRxzzEBOddfMwQcmUz5M-fYG&cnr=14&ratebypass=yes&dur=230.783&lmt=1629003351327319&mt=1635263580&fvip=4&fexp=24001373%2C24007246&c=WEB&txp=5532434&n=weByyA1lMexzGw&sparams=expire%2Cei%2Cip%2Cid%2Citag%2Csource%2Crequiressl%2Cvprv%2Cmime%2Cns%2Ccnr%2Cratebypass%2Cdur%2Clmt&sig=AOq0QJ8wRQIhANOlhWSb_gPfz92-pnpQb2PWdUsGf8bWBlLwpbpiYwDUAiBfO2TAr0fAa9KsqDbSwZT-XXVTSaLZ6eKI-10QmCCcBA%3D%3D&lsparams=mh%2Cmm%2Cmn%2Cms%2Cmv%2Cmvi%2Cpl%2Cinitcwndbps&lsig=AG3C_xAwRQIgJAnh8yMtmgsj_Qxioa10fzyWYYyTukU7NqcjTDfYqSICIQDLqSIdctJXexKXY2jtJCGxn-DQKKaAxg7AD1LHUdZYqA%3D%3D&title=Catch%20Me%20If%20You%20Can%3A%20Outsmarting%20the%20FBI%20(HD%20CLIP)",
		pathToSave:   "heavyrabbit.mp4",
		noOfSections: 10,
	}
	err := d.ourDownloader()
	if err != nil {
		log.Printf("Unable to download due to: %s\n", err)
	}
	fmt.Printf("Successfully Downloaded in %v seconds\n", time.Now().Sub(start).Seconds())
}

func (d File) merge(parts [][2]int) error {
	f, err := os.OpenFile(d.pathToSave, os.O_CREATE|os.O_WRONLY|os.O_APPEND, os.ModePerm)
	if err != nil {
		return err
	}
	defer f.Close()
	for i := range parts {
		tmpFileName := fmt.Sprintf("section-%v.tmp", i)
		b, err := ioutil.ReadFile(tmpFileName)
		if err != nil {
			return err
		}
		n, err := f.Write(b)
		if err != nil {
			return err
		}
		err = os.Remove(tmpFileName)
		if err != nil {
			return err
		}
		fmt.Printf("%v bytes merged\n", n)
	}
	return nil
}

func (d File) createRequest(method string) (*http.Request, error) {
	r, err := http.NewRequest(
		method,
		d.url,
		nil,
	)
	if err != nil {
		return nil, err
	}
	return r, nil
}

// Golang Routines will be calling this function to download concurrently
func (d File) ourDownloader() error {
	fmt.Println("Checking URL")
	r, err := d.createRequest("HEAD")
	if err != nil {
		return err
	}
	response, err := http.DefaultClient.Do(r)
	if err != nil {
		return err
	}
	fmt.Printf("Got %v\n", response.StatusCode)

	if response.StatusCode > 299 {
		return errors.New(fmt.Sprintf("Can't process, response is %v", response.StatusCode))
	}

	size, err := strconv.Atoi(response.Header.Get("Content-Length"))
	if err != nil {
		return err
	}
	fmt.Printf("Size is %v bytes\n", size)

	var parts = make([][2]int, d.noOfSections)
	eachSize := size / d.noOfSections

	for i := range parts {
		if i == 0 {
			parts[i][0] = 0
		} else {
			parts[i][0] = parts[i-1][1] + 1
		}

		if i < d.noOfSections-1 {
			parts[i][1] = parts[i][0] + eachSize
		} else {
			parts[i][1] = size - 1
		}
	}

	log.Println(parts)
	var wg sync.WaitGroup
	for i, s := range parts {
		wg.Add(1)
		go func(i int, s [2]int) {
			defer wg.Done()
			err = d.sectionDownloader(i, s)
			if err != nil {
				panic(err)
			}
		}(i, s)
	}
	wg.Wait()

	return d.merge(parts)
}

func (d File) sectionDownloader(i int, c [2]int) error {
	r, err := d.createRequest("GET")
	if err != nil {
		return err
	}
	r.Header.Set("Range", fmt.Sprintf("bytes=%v-%v", c[0], c[1]))
	response, err := http.DefaultClient.Do(r)
	if err != nil {
		return err
	}
	if response.StatusCode > 299 {
		return errors.New(fmt.Sprintf("Can't process, response is %v", response.StatusCode))
	}
	fmt.Printf("Downloaded %v bytes for section %v\n", response.Header.Get("Content-Length"), i)
	b, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(fmt.Sprintf("section-%v.tmp", i), b, os.ModePerm)
	if err != nil {
		return err
	}
	return nil
}
