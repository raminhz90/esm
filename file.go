package main

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"sync"

	"github.com/cheggaaa/pb"
	log "github.com/cihub/seelog"
)

func checkFileIsExist(filename string) bool {
	var exist = true
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		exist = false
	}
	return exist
}

func (m *Migrator) NewFileReadWorker(pb *pb.ProgressBar, wg *sync.WaitGroup) {
	log.Debug("start reading file")
	f, err := os.Open(m.Config.DumpInputFile)
	if err != nil {
		log.Error(err)
		return
	}

	defer f.Close()
	r := bufio.NewReader(f)
	lineCount := 0
	for {
		line, err := r.ReadString('\n')
		if io.EOF == err || nil != err {
			break
		}
		lineCount += 1
		js := map[string]interface{}{}

		err = DecodeJson(line, &js)
		if err != nil {
			log.Error(err)
			continue
		}
		m.DocChan <- js
		pb.Increment()
	}

	defer f.Close()
	log.Debug("end reading file")
	close(m.DocChan)
	wg.Done()
}

func (c *Migrator) NewFileDumpWorker(pb *pb.ProgressBar, wg *sync.WaitGroup) {
	var f *os.File
	var err1 error

	if checkFileIsExist(c.Config.DumpOutFile) {
		f, err1 = os.OpenFile(c.Config.DumpOutFile, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
		if err1 != nil {
			log.Error(err1)
			return
		}

	} else {
		f, err1 = os.Create(c.Config.DumpOutFile)
		if err1 != nil {
			log.Error(err1)
			return
		}
	}

	w := bufio.NewWriter(f)

READ_DOCS:
	for {
		docI, open := <-c.DocChan
		// this check is in case the document is an error with scroll stuff
		if status, ok := docI["status"]; ok {
			if status.(int) == 404 {
				log.Error("error: ", docI["response"])
				continue
			}
		}

		// sanity check
		for _, key := range []string{"_index", "_source", "_id"} {
			if _, ok := docI[key]; !ok {
				break READ_DOCS
			}
		}

		jsr, err := json.Marshal(docI)
		log.Trace(string(jsr))
		if err != nil {
			log.Error(err)
		}
		n, err := w.WriteString(string(jsr))
		if err != nil {
			log.Error(n, err)
		}
		w.WriteString("\n")
		pb.Increment()

		// if channel is closed flush and gtfo
		if !open {
			goto WORKER_DONE
		}
	}

WORKER_DONE:
	w.Flush()
	f.Close()

	wg.Done()
	log.Debug("file dump finished")
}
